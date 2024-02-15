package chainlink

import (
	"context"
	"errors"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	evmrelay "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
	"github.com/smartcontractkit/chainlink/v2/plugins"
)

type RelayerFactory struct {
	logger.Logger
	*plugins.LoopRegistry
	loop.GRPCOpts
}

type EVMFactoryConfig struct {
	legacyevm.ChainOpts
	evmrelay.CSAETHKeystore
}

func (r *RelayerFactory) NewEVM(ctx context.Context, config EVMFactoryConfig) (map[relay.ID]evmrelay.LoopRelayAdapter, error) {
	// TODO impl EVM loop. For now always 'fallback' to an adapter and embedded chain

	relayers := make(map[relay.ID]evmrelay.LoopRelayAdapter)

	lggr := r.Logger.Named("EVM")

	// override some common opts with the factory values. this seems weird... maybe other signatures should change, or this should take a different type...
	ccOpts := legacyevm.ChainRelayExtenderConfig{
		Logger:    lggr,
		KeyStore:  config.CSAETHKeystore.Eth(),
		ChainOpts: config.ChainOpts,
	}

	evmRelayExtenders, err := evmrelay.NewChainRelayerExtenders(ctx, ccOpts)
	if err != nil {
		return nil, err
	}
	legacyChains := evmrelay.NewLegacyChainsFromRelayerExtenders(evmRelayExtenders)
	for _, ext := range evmRelayExtenders.Slice() {
		relayID := relay.ID{Network: relay.EVM, ChainID: ext.Chain().ID().String()}
		chain, err2 := legacyChains.Get(relayID.ChainID)
		if err2 != nil {
			return nil, err2
		}

		relayerOpts := evmrelay.RelayerOpts{
			DB:               ccOpts.DB,
			QConfig:          ccOpts.AppConfig.Database(),
			CSAETHKeystore:   config.CSAETHKeystore,
			EventBroadcaster: ccOpts.EventBroadcaster,
		}
		relayer, err2 := evmrelay.NewRelayer(lggr.Named(relayID.ChainID), chain, relayerOpts)
		if err2 != nil {
			err = errors.Join(err, err2)
			continue
		}

		relayers[relayID] = evmrelay.NewLoopRelayServerAdapter(relayer, ext)
	}

	// always return err because it is accumulating individual errors
	return relayers, err
}
