package vrf

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/jmoiron/sqlx"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/utils/mailbox"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/assets"
	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/batch_vrf_coordinator_v2"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrf_coordinator_v2"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrf_owner"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore"
	"github.com/smartcontractkit/chainlink/v2/core/services/pg"
	"github.com/smartcontractkit/chainlink/v2/core/services/pipeline"
	v2 "github.com/smartcontractkit/chainlink/v2/core/services/vrf/v2"
	"github.com/smartcontractkit/chainlink/v2/core/services/vrf/vrfcommon"
)

type Delegate struct {
	q            pg.Q
	pr           pipeline.Runner
	porm         pipeline.ORM
	ks           keystore.Master
	legacyChains legacyevm.LegacyChainContainer
	lggr         logger.Logger
	mailMon      *mailbox.Monitor
}

func NewDelegate(
	db *sqlx.DB,
	ks keystore.Master,
	pr pipeline.Runner,
	porm pipeline.ORM,
	legacyChains legacyevm.LegacyChainContainer,
	lggr logger.Logger,
	cfg pg.QConfig,
	mailMon *mailbox.Monitor) *Delegate {
	return &Delegate{
		q:            pg.NewQ(db, lggr, cfg),
		ks:           ks,
		pr:           pr,
		porm:         porm,
		legacyChains: legacyChains,
		lggr:         lggr.Named("VRF"),
		mailMon:      mailMon,
	}
}

func (d *Delegate) JobType() job.Type {
	return job.VRF
}

func (d *Delegate) BeforeJobCreated(job.Job)              {}
func (d *Delegate) AfterJobCreated(job.Job)               {}
func (d *Delegate) BeforeJobDeleted(job.Job)              {}
func (d *Delegate) OnDeleteJob(job.Job, pg.Queryer) error { return nil }

// ServicesForSpec satisfies the job.Delegate interface.
func (d *Delegate) ServicesForSpec(jb job.Job) ([]job.ServiceCtx, error) {
	if jb.VRFSpec == nil || jb.PipelineSpec == nil {
		return nil, errors.Errorf("vrf.Delegate expects a VRFSpec and PipelineSpec to be present, got %+v", jb)
	}
	pl, err := jb.PipelineSpec.Pipeline()
	if err != nil {
		return nil, err
	}
	chain, err := d.legacyChains.Get(jb.VRFSpec.EVMChainID.String())
	if err != nil {
		return nil, err
	}
	coordinatorV2, err := vrf_coordinator_v2.NewVRFCoordinatorV2(jb.VRFSpec.CoordinatorAddress.Address(), chain.Client())
	if err != nil {
		return nil, err
	}

	// If the batch coordinator address is not provided, we will fall back to non-batched
	var batchCoordinatorV2 *batch_vrf_coordinator_v2.BatchVRFCoordinatorV2
	if jb.VRFSpec.BatchCoordinatorAddress != nil {
		batchCoordinatorV2, err = batch_vrf_coordinator_v2.NewBatchVRFCoordinatorV2(
			jb.VRFSpec.BatchCoordinatorAddress.Address(), chain.Client())
		if err != nil {
			return nil, errors.Wrap(err, "create batch coordinator wrapper")
		}
	}

	var vrfOwner *vrf_owner.VRFOwner
	if jb.VRFSpec.VRFOwnerAddress != nil {
		vrfOwner, err = vrf_owner.NewVRFOwner(
			jb.VRFSpec.VRFOwnerAddress.Address(), chain.Client(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "create vrf owner wrapper")
		}
	}

	l := d.lggr.Named(jb.ExternalJobID.String()).With(
		"jobID", jb.ID,
		"externalJobID", jb.ExternalJobID,
		"coordinatorAddress", jb.VRFSpec.CoordinatorAddress,
	)
	lV2 := l.Named("VRFListenerV2")

	for _, task := range pl.Tasks {
		if _, ok := task.(*pipeline.VRFTaskV2); ok {
			if err2 := CheckFromAddressesExist(jb, d.ks.Eth()); err != nil {
				return nil, err2
			}

			if !FromAddressMaxGasPricesAllEqual(jb, chain.Config().EVM().GasEstimator().PriceMaxKey) {
				return nil, errors.New("key-specific max gas prices of all fromAddresses are not equal, please set them to equal values")
			}

			if err2 := CheckFromAddressMaxGasPrices(jb, chain.Config().EVM().GasEstimator().PriceMaxKey); err != nil {
				return nil, err2
			}

			// Get the LINKETHFEED address with retries
			// This is needed because the RPC endpoint may be down so we need to
			// switch over to another one.
			// var linkEthFeedAddress common.Address
			// err = retry.Do(func() error {
			// 	linkEthFeedAddress, err = coordinatorV2.LINKETHFEED(nil)
			// 	return err
			// }, retry.Attempts(10), retry.Delay(500*time.Millisecond))
			// if err != nil {
			// 	return nil, errors.Wrap(err, "LINKETHFEED")
			// }
			// aggregator, err := aggregator_v3_interface.NewAggregatorV3Interface(linkEthFeedAddress, chain.Client())
			// if err != nil {
			// 	return nil, errors.Wrap(err, "NewAggregatorV3Interface")
			// }
			// if vrfOwner == nil {
			// 	lV2.Infow("Running without VRFOwnerAddress set on the spec")
			// }

			return []job.ServiceCtx{v2.New(
				chain.Config().EVM(),
				chain.Config().EVM().GasEstimator(),
				lV2,
				chain,
				chain.ID(),
				d.q,
				v2.NewCoordinatorV2(coordinatorV2),
				batchCoordinatorV2,
				vrfOwner,
				// aggregator,
				d.pr,
				d.ks.Eth(),
				jb,
				func() {},
				// the lookback in the deduper must be >= the lookback specified for the log poller
				// otherwise we will end up re-delivering logs that were already delivered.
				vrfcommon.NewInflightCache(int(chain.Config().EVM().FinalityDepth())),
				vrfcommon.NewLogDeduper(int(chain.Config().EVM().FinalityDepth())),
			),
			}, nil
		}
	}
	return nil, errors.New("invalid job spec expected a vrf task")
}

// CheckFromAddressesExist returns an error if and only if one of the addresses
// in the VRF spec's fromAddresses field does not exist in the keystore.
func CheckFromAddressesExist(jb job.Job, gethks keystore.Eth) (err error) {
	for _, a := range jb.VRFSpec.FromAddresses {
		_, err2 := gethks.Get(a.Hex())
		err = multierr.Append(err, err2)
	}
	return
}

// CheckFromAddressMaxGasPrices checks if the provided gas price in the job spec gas lane parameter
// matches what is set for the  provided from addresses.
// If they don't match, this is a configuration error. An error is returned with all the keys that do
// not match the provided gas lane price.
func CheckFromAddressMaxGasPrices(jb job.Job, keySpecificMaxGas keySpecificMaxGasFn) (err error) {
	if jb.VRFSpec.GasLanePrice != nil {
		for _, a := range jb.VRFSpec.FromAddresses {
			if keySpecific := keySpecificMaxGas(a.Address()); !keySpecific.Equal(jb.VRFSpec.GasLanePrice) {
				err = multierr.Append(err,
					fmt.Errorf(
						"key-specific max gas price of from address %s (%s) does not match gasLanePriceGWei (%s) specified in job spec",
						a.Hex(), keySpecific.String(), jb.VRFSpec.GasLanePrice.String()))
			}
		}
	}
	return
}

type keySpecificMaxGasFn func(common.Address) *assets.Wei

// FromAddressMaxGasPricesAllEqual returns true if and only if all the specified from
// addresses in the fromAddresses field of the VRF v2 job have the same key-specific max
// gas price.
func FromAddressMaxGasPricesAllEqual(jb job.Job, keySpecificMaxGasPriceWei keySpecificMaxGasFn) (allEqual bool) {
	allEqual = true
	for i := range jb.VRFSpec.FromAddresses {
		allEqual = allEqual && keySpecificMaxGasPriceWei(jb.VRFSpec.FromAddresses[i].Address()).Equal(
			keySpecificMaxGasPriceWei(jb.VRFSpec.FromAddresses[0].Address()),
		)
	}
	return
}
