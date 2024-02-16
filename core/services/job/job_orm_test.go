package job_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/bridges"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/assets"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/cltest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/configtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/pgtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/logger"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/keystore/keys/ethkey"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pipeline"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/vrf/vrfcommon"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/testdata/testspecs"
)

func TestORM_CreateJob_VRFV2(t *testing.T) {
	config := configtest.NewTestGeneralConfig(t)
	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, config.Database())

	lggr := logger.TestLogger(t)
	pipelineORM := pipeline.NewORM(db, lggr, config.Database(), config.JobPipeline().MaxSuccessfulRuns())
	bridgesORM := bridges.NewORM(db, lggr, config.Database())

	jobORM := NewTestORM(t, db, pipelineORM, bridgesORM, keyStore, config.Database())

	fromAddresses := []string{cltest.NewEIP55Address().String(), cltest.NewEIP55Address().String()}
	jb, err := vrfcommon.ValidatedVRFSpec(testspecs.GenerateVRFSpec(
		testspecs.VRFSpecParams{
			RequestedConfsDelay: 10,
			FromAddresses:       fromAddresses,
			ChunkSize:           25,
			BackoffInitialDelay: time.Minute,
			BackoffMaxDelay:     time.Hour,
			GasLanePrice:        assets.GWei(100),
			VRFOwnerAddress:     "0x32891BD79647DC9136Fc0a59AAB48c7825eb624c",
		}).
		Toml())
	require.NoError(t, err)

	require.NoError(t, jobORM.CreateJob(&jb))
	cltest.AssertCount(t, db, "vrf_specs", 1)
	cltest.AssertCount(t, db, "jobs", 1)
	var requestedConfsDelay int64
	require.NoError(t, db.Get(&requestedConfsDelay, `SELECT requested_confs_delay FROM vrf_specs LIMIT 1`))
	require.Equal(t, int64(10), requestedConfsDelay)
	var batchFulfillmentEnabled bool
	require.NoError(t, db.Get(&batchFulfillmentEnabled, `SELECT batch_fulfillment_enabled FROM vrf_specs LIMIT 1`))
	require.False(t, batchFulfillmentEnabled)
	var customRevertsPipelineEnabled bool
	require.NoError(t, db.Get(&customRevertsPipelineEnabled, `SELECT custom_reverts_pipeline_enabled FROM vrf_specs LIMIT 1`))
	require.False(t, customRevertsPipelineEnabled)
	var batchFulfillmentGasMultiplier float64
	require.NoError(t, db.Get(&batchFulfillmentGasMultiplier, `SELECT batch_fulfillment_gas_multiplier FROM vrf_specs LIMIT 1`))
	require.Equal(t, float64(1.0), batchFulfillmentGasMultiplier)
	var requestTimeout time.Duration
	require.NoError(t, db.Get(&requestTimeout, `SELECT request_timeout FROM vrf_specs LIMIT 1`))
	require.Equal(t, 24*time.Hour, requestTimeout)
	var backoffInitialDelay time.Duration
	require.NoError(t, db.Get(&backoffInitialDelay, `SELECT backoff_initial_delay FROM vrf_specs LIMIT 1`))
	require.Equal(t, time.Minute, backoffInitialDelay)
	var backoffMaxDelay time.Duration
	require.NoError(t, db.Get(&backoffMaxDelay, `SELECT backoff_max_delay FROM vrf_specs LIMIT 1`))
	require.Equal(t, time.Hour, backoffMaxDelay)
	var chunkSize int
	require.NoError(t, db.Get(&chunkSize, `SELECT chunk_size FROM vrf_specs LIMIT 1`))
	require.Equal(t, 25, chunkSize)
	var gasLanePrice assets.Wei
	require.NoError(t, db.Get(&gasLanePrice, `SELECT gas_lane_price FROM vrf_specs LIMIT 1`))
	require.Equal(t, jb.VRFSpec.GasLanePrice, &gasLanePrice)
	var fa pq.ByteaArray
	require.NoError(t, db.Get(&fa, `SELECT from_addresses FROM vrf_specs LIMIT 1`))
	var actual []string
	for _, b := range fa {
		actual = append(actual, common.BytesToAddress(b).String())
	}
	require.ElementsMatch(t, fromAddresses, actual)
	var vrfOwnerAddress ethkey.EIP55Address
	require.NoError(t, db.Get(&vrfOwnerAddress, `SELECT vrf_owner_address FROM vrf_specs LIMIT 1`))
	require.Equal(t, "0x32891BD79647DC9136Fc0a59AAB48c7825eb624c", vrfOwnerAddress.Address().String())
	require.NoError(t, jobORM.DeleteJob(jb.ID))
	cltest.AssertCount(t, db, "vrf_specs", 0)
	cltest.AssertCount(t, db, "jobs", 0)

	jb, err = vrfcommon.ValidatedVRFSpec(testspecs.GenerateVRFSpec(testspecs.VRFSpecParams{RequestTimeout: 1 * time.Hour}).Toml())
	require.NoError(t, err)
	require.NoError(t, jobORM.CreateJob(&jb))
	cltest.AssertCount(t, db, "vrf_specs", 1)
	cltest.AssertCount(t, db, "jobs", 1)
	require.NoError(t, db.Get(&requestedConfsDelay, `SELECT requested_confs_delay FROM vrf_specs LIMIT 1`))
	require.Equal(t, int64(0), requestedConfsDelay)
	require.NoError(t, db.Get(&requestTimeout, `SELECT request_timeout FROM vrf_specs LIMIT 1`))
	require.Equal(t, 1*time.Hour, requestTimeout)
	require.NoError(t, jobORM.DeleteJob(jb.ID))
	cltest.AssertCount(t, db, "vrf_specs", 0)
	cltest.AssertCount(t, db, "jobs", 0)
}

func TestORM_CreateJob_EVMChainID_Validation(t *testing.T) {
	config := configtest.NewGeneralConfig(t, nil)
	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, config.Database())

	lggr := logger.TestLogger(t)
	pipelineORM := pipeline.NewORM(db, lggr, config.Database(), config.JobPipeline().MaxSuccessfulRuns())
	bridgesORM := bridges.NewORM(db, lggr, config.Database())

	jobORM := NewTestORM(t, db, pipelineORM, bridgesORM, keyStore, config.Database())

	t.Run("evm chain id validation for vrf works", func(t *testing.T) {
		jb := job.Job{
			Type:    job.VRF,
			VRFSpec: &job.VRFSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for block hash store works", func(t *testing.T) {
		jb := job.Job{
			Type:               job.BlockhashStore,
			BlockhashStoreSpec: &job.BlockhashStoreSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for block header feeder works", func(t *testing.T) {
		jb := job.Job{
			Type:                  job.BlockHeaderFeeder,
			BlockHeaderFeederSpec: &job.BlockHeaderFeederSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for legacy gas station server spec works", func(t *testing.T) {
		jb := job.Job{
			Type:                       job.LegacyGasStationServer,
			LegacyGasStationServerSpec: &job.LegacyGasStationServerSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for legacy gas station sidecar spec works", func(t *testing.T) {
		jb := job.Job{
			Type:                        job.LegacyGasStationSidecar,
			LegacyGasStationSidecarSpec: &job.LegacyGasStationSidecarSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})
}

func mustInsertPipelineRun(t *testing.T, orm pipeline.ORM, j job.Job) pipeline.Run {
	t.Helper()

	run := pipeline.Run{
		PipelineSpecID: j.PipelineSpecID,
		State:          pipeline.RunStatusRunning,
		Outputs:        pipeline.JSONSerializable{Valid: false},
		AllErrors:      pipeline.RunErrors{},
		CreatedAt:      time.Now(),
		FinishedAt:     null.Time{},
	}
	err := orm.CreateRun(&run)
	require.NoError(t, err)
	return run
}
