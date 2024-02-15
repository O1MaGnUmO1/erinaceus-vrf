package job_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink/v2/core/bridges"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/assets"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/configtest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/v2/core/services/vrf/vrfcommon"
	"github.com/smartcontractkit/chainlink/v2/core/services/webhook"
	"github.com/smartcontractkit/chainlink/v2/core/testdata/testspecs"
)

const mercuryOracleTOML = `name = 'LINK / ETH | 0x0000000000000000000000000000000000000000000000000000000000000001 | verifier_proxy 0x0000000000000000000000000000000000000001'
type = 'offchainreporting2'
schemaVersion = 1
externalJobID = '00000000-0000-0000-0000-000000000001'
contractID = '0x0000000000000000000000000000000000000006'
transmitterID = '%s'
feedID = '%s'
relay = 'evm'
pluginType = 'mercury'
observationSource = """
	ds          [type=http method=GET url="https://chain.link/ETH-USD"];
	ds_parse    [type=jsonparse path="data.price" separator="."];
	ds_multiply [type=multiply times=100];
	ds -> ds_parse -> ds_multiply;
"""

[relayConfig]
chainID = 1
fromBlock = 1000

[pluginConfig]
serverURL = 'wss://localhost:8080'
serverPubKey = '8fa807463ad73f9ee855cfd60ba406dcf98a2855b3dd8af613107b0f6890a707'
`

func TestORM(t *testing.T) {
	t.Parallel()
	config := configtest.NewTestGeneralConfig(t)
	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, config.Database())
	ethKeyStore := keyStore.Eth()

	require.NoError(t, keyStore.P2P().Add(cltest.DefaultP2PKey))

	pipelineORM := pipeline.NewORM(db, logger.TestLogger(t), config.Database(), config.JobPipeline().MaxSuccessfulRuns())
	bridgesORM := bridges.NewORM(db, logger.TestLogger(t), config.Database())

	orm := NewTestORM(t, db, pipelineORM, bridgesORM, keyStore, config.Database())
	borm := bridges.NewORM(db, logger.TestLogger(t), config.Database())

	_, bridge := cltest.MustCreateBridge(t, db, cltest.BridgeOpts{}, config.Database())
	_, bridge2 := cltest.MustCreateBridge(t, db, cltest.BridgeOpts{}, config.Database())
	_, address := cltest.MustInsertRandomKey(t, ethKeyStore)
	jb := makeOCRJobSpec(t, address, bridge.Name.String(), bridge2.Name.String())

	t.Run("it creates job specs", func(t *testing.T) {
		err := orm.CreateJob(jb)
		require.NoError(t, err)

		var returnedSpec job.Job
		var OCROracleSpec job.OCROracleSpec

		err = db.Get(&returnedSpec, "SELECT * FROM jobs WHERE jobs.id = $1", jb.ID)
		require.NoError(t, err)
		err = db.Get(&OCROracleSpec, "SELECT * FROM ocr_oracle_specs WHERE ocr_oracle_specs.id = $1", jb.OCROracleSpecID)
		require.NoError(t, err)
		returnedSpec.OCROracleSpec = &OCROracleSpec
		compareOCRJobSpecs(t, *jb, returnedSpec)
	})

	t.Run("autogenerates external job ID if missing", func(t *testing.T) {
		jb2 := makeOCRJobSpec(t, address, bridge.Name.String(), bridge2.Name.String())
		jb2.ExternalJobID = uuid.UUID{}
		err := orm.CreateJob(jb2)
		require.NoError(t, err)

		var returnedSpec job.Job
		err = db.Get(&returnedSpec, "SELECT * FROM jobs WHERE jobs.id = $1", jb.ID)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.UUID{}, returnedSpec.ExternalJobID)
	})

	t.Run("it deletes jobs from the DB", func(t *testing.T) {
		var dbSpecs []job.Job

		err := db.Select(&dbSpecs, "SELECT * FROM jobs")
		require.NoError(t, err)
		require.Len(t, dbSpecs, 2)

		err = orm.DeleteJob(jb.ID)
		require.NoError(t, err)

		dbSpecs = []job.Job{}
		err = db.Select(&dbSpecs, "SELECT * FROM jobs")
		require.NoError(t, err)
		require.Len(t, dbSpecs, 1)
	})

	t.Run("increase job spec error occurrence", func(t *testing.T) {
		jb3 := makeOCRJobSpec(t, address, bridge.Name.String(), bridge2.Name.String())
		err := orm.CreateJob(jb3)
		require.NoError(t, err)
		var jobSpec job.Job
		err = db.Get(&jobSpec, "SELECT * FROM jobs")
		require.NoError(t, err)

		ocrSpecError1 := "ocr spec 1 errored"
		ocrSpecError2 := "ocr spec 2 errored"
		require.NoError(t, orm.RecordError(jobSpec.ID, ocrSpecError1))
		require.NoError(t, orm.RecordError(jobSpec.ID, ocrSpecError1))
		require.NoError(t, orm.RecordError(jobSpec.ID, ocrSpecError2))

		var specErrors []job.SpecError
		err = db.Select(&specErrors, "SELECT * FROM job_spec_errors")
		require.NoError(t, err)
		require.Len(t, specErrors, 2)

		assert.Equal(t, specErrors[0].Occurrences, uint(2))
		assert.Equal(t, specErrors[1].Occurrences, uint(1))
		assert.True(t, specErrors[0].CreatedAt.Before(specErrors[0].UpdatedAt), "expected created_at (%s) to be before updated_at (%s)", specErrors[0].CreatedAt, specErrors[0].UpdatedAt)
		assert.Equal(t, specErrors[0].Description, ocrSpecError1)
		assert.Equal(t, specErrors[1].Description, ocrSpecError2)
		assert.True(t, specErrors[1].CreatedAt.After(specErrors[0].UpdatedAt))
		var j2 job.Job
		var OCROracleSpec job.OCROracleSpec
		var jobSpecErrors []job.SpecError

		err = db.Get(&j2, "SELECT * FROM jobs WHERE jobs.id = $1", jobSpec.ID)
		require.NoError(t, err)
		err = db.Get(&OCROracleSpec, "SELECT * FROM ocr_oracle_specs WHERE ocr_oracle_specs.id = $1", j2.OCROracleSpecID)
		require.NoError(t, err)
		err = db.Select(&jobSpecErrors, "SELECT * FROM job_spec_errors WHERE job_spec_errors.job_id = $1", j2.ID)
		require.NoError(t, err)
		require.Len(t, jobSpecErrors, 2)
	})

	t.Run("finds job spec error by ID", func(t *testing.T) {
		jb3 := makeOCRJobSpec(t, address, bridge.Name.String(), bridge2.Name.String())
		err := orm.CreateJob(jb3)
		require.NoError(t, err)
		var jobSpec job.Job
		err = db.Get(&jobSpec, "SELECT * FROM jobs")
		require.NoError(t, err)

		var specErrors []job.SpecError
		err = db.Select(&specErrors, "SELECT * FROM job_spec_errors")
		require.NoError(t, err)
		require.Len(t, specErrors, 2)

		ocrSpecError1 := "ocr spec 3 errored"
		ocrSpecError2 := "ocr spec 4 errored"
		require.NoError(t, orm.RecordError(jobSpec.ID, ocrSpecError1))
		require.NoError(t, orm.RecordError(jobSpec.ID, ocrSpecError2))

		var updatedSpecError []job.SpecError

		err = db.Select(&updatedSpecError, "SELECT * FROM job_spec_errors ORDER BY id ASC")
		require.NoError(t, err)
		require.Len(t, updatedSpecError, 4)

		assert.Equal(t, uint(1), updatedSpecError[2].Occurrences)
		assert.Equal(t, uint(1), updatedSpecError[3].Occurrences)
		assert.Equal(t, ocrSpecError1, updatedSpecError[2].Description)
		assert.Equal(t, ocrSpecError2, updatedSpecError[3].Description)

		dbSpecErr1, err := orm.FindSpecError(updatedSpecError[2].ID)
		require.NoError(t, err)
		dbSpecErr2, err := orm.FindSpecError(updatedSpecError[3].ID)
		require.NoError(t, err)

		assert.Equal(t, uint(1), dbSpecErr1.Occurrences)
		assert.Equal(t, uint(1), dbSpecErr2.Occurrences)
		assert.Equal(t, ocrSpecError1, dbSpecErr1.Description)
		assert.Equal(t, ocrSpecError2, dbSpecErr2.Description)
	})

	t.Run("creates webhook specs along with external_initiator_webhook_specs", func(t *testing.T) {
		eiFoo := cltest.MustInsertExternalInitiator(t, borm)
		eiBar := cltest.MustInsertExternalInitiator(t, borm)

		eiWS := []webhook.TOMLWebhookSpecExternalInitiator{
			{Name: eiFoo.Name, Spec: cltest.JSONFromString(t, `{}`)},
			{Name: eiBar.Name, Spec: cltest.JSONFromString(t, `{"bar": 1}`)},
		}
		eim := webhook.NewExternalInitiatorManager(db, nil, logger.TestLogger(t), config.Database())
		jb, err := webhook.ValidatedWebhookSpec(testspecs.GenerateWebhookSpec(testspecs.WebhookSpecParams{ExternalInitiators: eiWS}).Toml(), eim)
		require.NoError(t, err)

		err = orm.CreateJob(&jb)
		require.NoError(t, err)

		cltest.AssertCount(t, db, "external_initiator_webhook_specs", 2)
	})

}

func TestORM_DeleteJob_DeletesAssociatedRecords(t *testing.T) {
	t.Parallel()
	config := configtest.NewGeneralConfig(t, nil)

	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, config.Database())
	require.NoError(t, keyStore.P2P().Add(cltest.DefaultP2PKey))

	lggr := logger.TestLogger(t)
	pipelineORM := pipeline.NewORM(db, lggr, config.Database(), config.JobPipeline().MaxSuccessfulRuns())
	bridgesORM := bridges.NewORM(db, lggr, config.Database())
	jobORM := NewTestORM(t, db, pipelineORM, bridgesORM, keyStore, config.Database())

	t.Run("it creates and deletes records for vrf jobs", func(t *testing.T) {
		key, err := keyStore.VRF().Create()
		require.NoError(t, err)
		pk := key.PublicKey
		jb, err := vrfcommon.ValidatedVRFSpec(testspecs.GenerateVRFSpec(testspecs.VRFSpecParams{PublicKey: pk.String()}).Toml())
		require.NoError(t, err)

		err = jobORM.CreateJob(&jb)
		require.NoError(t, err)
		cltest.AssertCount(t, db, "vrf_specs", 1)
		cltest.AssertCount(t, db, "jobs", 1)
		err = jobORM.DeleteJob(jb.ID)
		require.NoError(t, err)
		cltest.AssertCount(t, db, "vrf_specs", 0)
		cltest.AssertCount(t, db, "jobs", 0)
	})

	t.Run("it deletes records for webhook jobs", func(t *testing.T) {
		ei := cltest.MustInsertExternalInitiator(t, bridges.NewORM(db, logger.TestLogger(t), config.Database()))
		jb, webhookSpec := cltest.MustInsertWebhookSpec(t, db)
		_, err := db.Exec(`INSERT INTO external_initiator_webhook_specs (external_initiator_id, webhook_spec_id, spec) VALUES ($1,$2,$3)`, ei.ID, webhookSpec.ID, `{"ei": "foo", "name": "webhookSpecTwoEIs"}`)
		require.NoError(t, err)

		err = jobORM.DeleteJob(jb.ID)
		require.NoError(t, err)
		cltest.AssertCount(t, db, "webhook_specs", 0)
		cltest.AssertCount(t, db, "external_initiator_webhook_specs", 0)
		cltest.AssertCount(t, db, "jobs", 0)
	})

	t.Run("does not allow to delete external initiators if they have referencing external_initiator_webhook_specs", func(t *testing.T) {
		// create new db because this will rollback transaction and poison it
		db := pgtest.NewSqlxDB(t)
		ei := cltest.MustInsertExternalInitiator(t, bridges.NewORM(db, logger.TestLogger(t), config.Database()))
		_, webhookSpec := cltest.MustInsertWebhookSpec(t, db)
		_, err := db.Exec(`INSERT INTO external_initiator_webhook_specs (external_initiator_id, webhook_spec_id, spec) VALUES ($1,$2,$3)`, ei.ID, webhookSpec.ID, `{"ei": "foo", "name": "webhookSpecTwoEIs"}`)
		require.NoError(t, err)

		_, err = db.Exec(`DELETE FROM external_initiators`)
		require.EqualError(t, err, "ERROR: update or delete on table \"external_initiators\" violates foreign key constraint \"external_initiator_webhook_specs_external_initiator_id_fkey\" on table \"external_initiator_webhook_specs\" (SQLSTATE 23503)")
	})
}

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

func TestORM_CreateJob_VRFV2Plus(t *testing.T) {
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
			VRFVersion:                   vrfcommon.V2Plus,
			RequestedConfsDelay:          10,
			FromAddresses:                fromAddresses,
			ChunkSize:                    25,
			BackoffInitialDelay:          time.Minute,
			BackoffMaxDelay:              time.Hour,
			GasLanePrice:                 assets.GWei(100),
			CustomRevertsPipelineEnabled: true,
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
	require.True(t, customRevertsPipelineEnabled)
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
	require.Error(t, db.Get(&vrfOwnerAddress, `SELECT vrf_owner_address FROM vrf_specs LIMIT 1`))
	require.NoError(t, jobORM.DeleteJob(jb.ID))
	cltest.AssertCount(t, db, "vrf_specs", 0)
	cltest.AssertCount(t, db, "jobs", 0)

	jb, err = vrfcommon.ValidatedVRFSpec(testspecs.GenerateVRFSpec(testspecs.VRFSpecParams{
		VRFVersion:     vrfcommon.V2Plus,
		RequestTimeout: 1 * time.Hour,
		FromAddresses:  fromAddresses,
	}).Toml())
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

	t.Run("evm chain id validation for ocr works", func(t *testing.T) {
		jb := job.Job{
			Type:          job.OffchainReporting,
			OCROracleSpec: &job.OCROracleSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for direct request works", func(t *testing.T) {
		jb := job.Job{
			Type:              job.DirectRequest,
			DirectRequestSpec: &job.DirectRequestSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for flux monitor works", func(t *testing.T) {
		jb := job.Job{
			Type:            job.FluxMonitor,
			FluxMonitorSpec: &job.FluxMonitorSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

	t.Run("evm chain id validation for keepers works", func(t *testing.T) {
		jb := job.Job{
			Type:       job.Keeper,
			KeeperSpec: &job.KeeperSpec{},
		}
		assert.Equal(t, "CreateJobFailed: evm chain id must be defined", jobORM.CreateJob(&jb).Error())
	})

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
