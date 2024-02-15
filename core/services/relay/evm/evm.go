package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"

	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/services"
	commontypes "github.com/O1MaGnUmO1/chainlink-common/pkg/types"

	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/pg"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay/evm/types"
)

var _ commontypes.Relayer = &Relayer{} //nolint:staticcheck

type Relayer struct {
	db               *sqlx.DB
	chain            legacyevm.Chain
	lggr             logger.Logger
	ks               CSAETHKeystore
	eventBroadcaster pg.EventBroadcaster
	pgCfg            pg.QConfig
	chainReader      commontypes.ChainReader
}

type CSAETHKeystore interface {
	CSA() keystore.CSA
	Eth() keystore.Eth
}

type RelayerOpts struct {
	*sqlx.DB
	pg.QConfig
	CSAETHKeystore
	pg.EventBroadcaster
}

func (c RelayerOpts) Validate() error {
	var err error
	if c.DB == nil {
		err = errors.Join(err, errors.New("nil DB"))
	}
	if c.QConfig == nil {
		err = errors.Join(err, errors.New("nil QConfig"))
	}
	if c.CSAETHKeystore == nil {
		err = errors.Join(err, errors.New("nil Keystore"))
	}
	if c.EventBroadcaster == nil {
		err = errors.Join(err, errors.New("nil Eventbroadcaster"))
	}

	if err != nil {
		err = fmt.Errorf("invalid RelayerOpts: %w", err)
	}
	return err
}

func NewRelayer(lggr logger.Logger, chain legacyevm.Chain, opts RelayerOpts) (*Relayer, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("cannot create evm relayer: %w", err)
	}
	lggr = lggr.Named("Relayer")
	return &Relayer{
		db:               opts.DB,
		chain:            chain,
		lggr:             lggr,
		ks:               opts.CSAETHKeystore,
		eventBroadcaster: opts.EventBroadcaster,
		pgCfg:            opts.QConfig,
	}, nil
}

func (r *Relayer) Name() string {
	return r.lggr.Name()
}

// Start does noop: no subservices started on relay start, but when the first job is started
func (r *Relayer) Start(context.Context) error {
	return nil
}

func (r *Relayer) Close() error {
	return nil
}

// Ready does noop: always ready
func (r *Relayer) Ready() error {
	return r.chain.Ready()
}

func (r *Relayer) HealthReport() (report map[string]error) {
	report = make(map[string]error)
	maps.Copy(report, r.chain.HealthReport())
	return
}

func FilterNamesFromRelayArgs(args commontypes.RelayArgs) (filterNames []string, err error) {
	var addr ethkey.EIP55Address
	if addr, err = ethkey.NewEIP55Address(args.ContractID); err != nil {
		return nil, err
	}
	var relayConfig types.RelayConfig
	if err = json.Unmarshal(args.RelayConfig, &relayConfig); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	filterNames = []string{configPollerFilterName(addr.Address()), transmitterFilterName(addr.Address())}
	return filterNames, err
}

type configWatcher struct {
	services.StateMachine
	lggr             logger.Logger
	contractAddress  common.Address
	contractABI      abi.ABI
	offchainDigester ocrtypes.OffchainConfigDigester
	configPoller     types.ConfigPoller
	chain            legacyevm.Chain
	runReplay        bool
	fromBlock        uint64
	replayCtx        context.Context
	replayCancel     context.CancelFunc
	wg               sync.WaitGroup
}

func newConfigWatcher(lggr logger.Logger,
	contractAddress common.Address,
	contractABI abi.ABI,
	offchainDigester ocrtypes.OffchainConfigDigester,
	configPoller types.ConfigPoller,
	chain legacyevm.Chain,
	fromBlock uint64,
	runReplay bool,
) *configWatcher {
	replayCtx, replayCancel := context.WithCancel(context.Background())
	return &configWatcher{
		lggr:             lggr.Named("ConfigWatcher").Named(contractAddress.String()),
		contractAddress:  contractAddress,
		contractABI:      contractABI,
		offchainDigester: offchainDigester,
		configPoller:     configPoller,
		chain:            chain,
		runReplay:        runReplay,
		fromBlock:        fromBlock,
		replayCtx:        replayCtx,
		replayCancel:     replayCancel,
	}

}

func (c *configWatcher) Name() string {
	return c.lggr.Name()
}

func (c *configWatcher) Start(ctx context.Context) error {
	return c.StartOnce(fmt.Sprintf("configWatcher %x", c.contractAddress), func() error {
		if c.runReplay && c.fromBlock != 0 {
			// Only replay if it's a brand runReplay job.
			c.wg.Add(1)
			go func() {
				defer c.wg.Done()
				c.lggr.Infow("starting replay for config", "fromBlock", c.fromBlock)
				if err := c.configPoller.Replay(c.replayCtx, int64(c.fromBlock)); err != nil {
					c.lggr.Errorf("error replaying for config", "err", err)
				} else {
					c.lggr.Infow("completed replaying for config", "fromBlock", c.fromBlock)
				}
			}()
		}
		c.configPoller.Start()
		return nil
	})
}

func (c *configWatcher) Close() error {
	return c.StopOnce(fmt.Sprintf("configWatcher %x", c.contractAddress), func() error {
		c.replayCancel()
		c.wg.Wait()
		return c.configPoller.Close()
	})
}

func (c *configWatcher) HealthReport() map[string]error {
	return map[string]error{c.Name(): c.Healthy()}
}

func (c *configWatcher) OffchainConfigDigester() ocrtypes.OffchainConfigDigester {
	return c.offchainDigester
}

func (c *configWatcher) ContractConfigTracker() ocrtypes.ContractConfigTracker {
	return c.configPoller
}
