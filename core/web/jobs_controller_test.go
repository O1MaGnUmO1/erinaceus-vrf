package web_test

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/pelletier/go-toml"
	ragep2ptypes "github.com/smartcontractkit/libocr/ragep2p/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/utils"
	evmclimocks "github.com/smartcontractkit/chainlink/v2/core/chains/evm/client/mocks"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/configtest"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/pg"
	"github.com/smartcontractkit/chainlink/v2/core/testdata/testspecs"
	"github.com/smartcontractkit/chainlink/v2/core/web"
	"github.com/smartcontractkit/chainlink/v2/core/web/presenters"
)

func TestJobsController_Create_ValidationFailure_OffchainReportingSpec(t *testing.T) {
	var (
		contractAddress = cltest.NewEIP55Address()
	)

	var peerID ragep2ptypes.PeerID
	require.NoError(t, peerID.UnmarshalText([]byte(configtest.DefaultPeerID)))
	randomBytes := testutils.Random32Byte()

	var tt = []struct {
		name        string
		pid         p2pkey.PeerID
		kb          string
		taExists    bool
		expectedErr error
	}{
		{
			name:        "invalid keybundle",
			pid:         p2pkey.PeerID(peerID),
			kb:          hex.EncodeToString(randomBytes[:]),
			taExists:    true,
			expectedErr: job.ErrNoSuchKeyBundle,
		},
		{
			name:        "invalid transmitter address",
			pid:         p2pkey.PeerID(peerID),
			kb:          cltest.DefaultOCRKeyBundleID,
			taExists:    false,
			expectedErr: job.ErrNoSuchTransmitterKey,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ta, client := setupJobsControllerTests(t)

			var address ethkey.EIP55Address
			if tc.taExists {
				key, _ := cltest.MustInsertRandomKey(t, ta.KeyStore.Eth())
				address = key.EIP55Address
			} else {
				address = cltest.NewEIP55Address()
			}

			sp := cltest.MinimalOCRNonBootstrapSpec(contractAddress, address, tc.pid, tc.kb)
			body, _ := json.Marshal(web.CreateJobRequest{
				TOML: sp,
			})
			resp, cleanup := client.Post("/v2/jobs", bytes.NewReader(body))
			t.Cleanup(cleanup)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(b), tc.expectedErr.Error())
		})
	}
}

func mustInt32FromString(t *testing.T, s string) int32 {
	i, err := strconv.ParseInt(s, 10, 32)
	require.NoError(t, err)
	return int32(i)
}

func TestJobsController_Create_WebhookSpec(t *testing.T) {
	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start(testutils.Context(t)))
	t.Cleanup(func() { assert.NoError(t, app.Stop()) })

	_, fetchBridge := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())
	_, submitBridge := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())

	client := app.NewHTTPClient(nil)

	tomlStr := testspecs.GetWebhookSpecNoBody(uuid.New(), fetchBridge.Name.String(), submitBridge.Name.String())
	body, _ := json.Marshal(web.CreateJobRequest{
		TOML: tomlStr,
	})
	response, cleanup := client.Post("/v2/jobs", bytes.NewReader(body))
	defer cleanup()
	require.Equal(t, http.StatusOK, response.StatusCode)
	resource := presenters.JobResource{}
	err := web.ParseJSONAPIResponse(cltest.ParseResponseBody(t, response), &resource)
	require.NoError(t, err)
	assert.NotNil(t, resource.PipelineSpec.DotDAGSource)

	jorm := app.JobORM()
	_, err = jorm.FindJob(testutils.Context(t), mustInt32FromString(t, resource.ID))
	require.NoError(t, err)
}

//go:embed webhook-spec-template.yml
var webhookSpecTemplate string

func TestJobsController_FailToCreate_EmptyJsonAttribute(t *testing.T) {
	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start(testutils.Context(t)))

	client := app.NewHTTPClient(nil)

	nameAndExternalJobID := uuid.New()
	spec := fmt.Sprintf(webhookSpecTemplate, nameAndExternalJobID, nameAndExternalJobID)
	body, err := json.Marshal(web.CreateJobRequest{
		TOML: spec,
	})
	require.NoError(t, err)
	response, cleanup := client.Post("/v2/jobs", bytes.NewReader(body))
	defer cleanup()

	b, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Contains(t, string(b), "syntax is not supported. Please use \\\"{}\\\" instead")
}

func TestJobsController_Update_HappyPath(t *testing.T) {
	cfg := configtest.NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {
		c.OCR.Enabled = ptr(true)
		c.P2P.V2.Enabled = ptr(true)
		c.P2P.V2.ListenAddresses = &[]string{fmt.Sprintf("127.0.0.1:%d", freeport.GetOne(t))}
		c.P2P.PeerID = &cltest.DefaultP2PPeerID
	})
	app := cltest.NewApplicationWithConfigAndKey(t, cfg, cltest.DefaultP2PKey)

	require.NoError(t, app.Start(testutils.Context(t)))

	_, bridge := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())
	_, bridge2 := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())

	client := app.NewHTTPClient(nil)

	var jb job.Job
	ocrspec := testspecs.GenerateOCRSpec(testspecs.OCRSpecParams{
		DS1BridgeName: bridge.Name.String(),
		DS2BridgeName: bridge2.Name.String(),
		Name:          "old OCR job",
	})
	err := toml.Unmarshal([]byte(ocrspec.Toml()), &jb)
	require.NoError(t, err)

	// BCF-2095
	// disable fkey checks until the end of the test transaction
	require.NoError(t, utils.JustError(
		app.GetSqlxDB().Exec(`SET CONSTRAINTS job_spec_errors_v2_job_id_fkey DEFERRED`)))

	var ocrSpec job.OCROracleSpec
	err = toml.Unmarshal([]byte(ocrspec.Toml()), &ocrSpec)
	require.NoError(t, err)
	jb.OCROracleSpec = &ocrSpec
	jb.OCROracleSpec.TransmitterAddress = &app.Keys[0].EIP55Address
	err = app.AddJobV2(testutils.Context(t), &jb)
	require.NoError(t, err)
	dbJb, err := app.JobORM().FindJob(testutils.Context(t), jb.ID)
	require.NoError(t, err)
	require.Equal(t, dbJb.Name.String, ocrspec.Name)

	// test Calling update on the job id with changed values should succeed.
	updatedSpec := testspecs.GenerateOCRSpec(testspecs.OCRSpecParams{
		DS1BridgeName:      bridge2.Name.String(),
		DS2BridgeName:      bridge.Name.String(),
		Name:               "updated OCR job",
		TransmitterAddress: app.Keys[0].Address.Hex(),
	})
	require.NoError(t, err)
	body, _ := json.Marshal(web.UpdateJobRequest{
		TOML: updatedSpec.Toml(),
	})
	response, cleanup := client.Put("/v2/jobs/"+fmt.Sprintf("%v", jb.ID), bytes.NewReader(body))
	t.Cleanup(cleanup)

	dbJb, err = app.JobORM().FindJob(testutils.Context(t), jb.ID)
	require.NoError(t, err)
	require.Equal(t, dbJb.Name.String, updatedSpec.Name)

	cltest.AssertServerResponse(t, response, http.StatusOK)
}

func TestJobsController_Update_NonExistentID(t *testing.T) {
	cfg := configtest.NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {
		c.OCR.Enabled = ptr(true)
		c.P2P.V2.Enabled = ptr(true)
		c.P2P.V2.ListenAddresses = &[]string{fmt.Sprintf("127.0.0.1:%d", freeport.GetOne(t))}
		c.P2P.PeerID = &cltest.DefaultP2PPeerID
	})
	app := cltest.NewApplicationWithConfigAndKey(t, cfg, cltest.DefaultP2PKey)

	require.NoError(t, app.Start(testutils.Context(t)))

	_, bridge := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())
	_, bridge2 := cltest.MustCreateBridge(t, app.GetSqlxDB(), cltest.BridgeOpts{}, app.GetConfig().Database())

	client := app.NewHTTPClient(nil)

	var jb job.Job
	ocrspec := testspecs.GenerateOCRSpec(testspecs.OCRSpecParams{
		DS1BridgeName: bridge.Name.String(),
		DS2BridgeName: bridge2.Name.String(),
		Name:          "old OCR job",
	})
	err := toml.Unmarshal([]byte(ocrspec.Toml()), &jb)
	require.NoError(t, err)
	var ocrSpec job.OCROracleSpec
	err = toml.Unmarshal([]byte(ocrspec.Toml()), &ocrSpec)
	require.NoError(t, err)
	jb.OCROracleSpec = &ocrSpec
	jb.OCROracleSpec.TransmitterAddress = &app.Keys[0].EIP55Address
	err = app.AddJobV2(testutils.Context(t), &jb)
	require.NoError(t, err)

	// test Calling update on the job id with changed values should succeed.
	updatedSpec := testspecs.GenerateOCRSpec(testspecs.OCRSpecParams{
		DS1BridgeName:      bridge2.Name.String(),
		DS2BridgeName:      bridge.Name.String(),
		Name:               "updated OCR job",
		TransmitterAddress: app.Keys[0].EIP55Address.String(),
	})
	require.NoError(t, err)
	body, _ := json.Marshal(web.UpdateJobRequest{
		TOML: updatedSpec.Toml(),
	})
	response, cleanup := client.Put("/v2/jobs/99999", bytes.NewReader(body))
	t.Cleanup(cleanup)
	cltest.AssertServerResponse(t, response, http.StatusNotFound)
}

func runOCRJobSpecAssertions(t *testing.T, ocrJobSpecFromFileDB job.Job, ocrJobSpecFromServer presenters.JobResource) {
	ocrJobSpecFromFile := ocrJobSpecFromFileDB.OCROracleSpec
	assert.Equal(t, ocrJobSpecFromFile.ContractAddress, ocrJobSpecFromServer.OffChainReportingSpec.ContractAddress)
	assert.Equal(t, ocrJobSpecFromFile.P2PV2Bootstrappers, ocrJobSpecFromServer.OffChainReportingSpec.P2PV2Bootstrappers)
	assert.Equal(t, ocrJobSpecFromFile.IsBootstrapPeer, ocrJobSpecFromServer.OffChainReportingSpec.IsBootstrapPeer)
	assert.Equal(t, ocrJobSpecFromFile.EncryptedOCRKeyBundleID, ocrJobSpecFromServer.OffChainReportingSpec.EncryptedOCRKeyBundleID)
	assert.Equal(t, ocrJobSpecFromFile.TransmitterAddress, ocrJobSpecFromServer.OffChainReportingSpec.TransmitterAddress)
	assert.Equal(t, ocrJobSpecFromFile.ObservationTimeout, ocrJobSpecFromServer.OffChainReportingSpec.ObservationTimeout)
	assert.Equal(t, ocrJobSpecFromFile.BlockchainTimeout, ocrJobSpecFromServer.OffChainReportingSpec.BlockchainTimeout)
	assert.Equal(t, ocrJobSpecFromFile.ContractConfigTrackerSubscribeInterval, ocrJobSpecFromServer.OffChainReportingSpec.ContractConfigTrackerSubscribeInterval)
	assert.Equal(t, ocrJobSpecFromFile.ContractConfigTrackerSubscribeInterval, ocrJobSpecFromServer.OffChainReportingSpec.ContractConfigTrackerSubscribeInterval)
	assert.Equal(t, ocrJobSpecFromFile.ContractConfigConfirmations, ocrJobSpecFromServer.OffChainReportingSpec.ContractConfigConfirmations)
	assert.Equal(t, ocrJobSpecFromFileDB.Pipeline.Source, ocrJobSpecFromServer.PipelineSpec.DotDAGSource)

	// Check that create and update dates are non empty values.
	// Empty date value is "0001-01-01 00:00:00 +0000 UTC" so we are checking for the
	// millennia and century characters to be present
	assert.Contains(t, ocrJobSpecFromServer.OffChainReportingSpec.CreatedAt.String(), "20")
	assert.Contains(t, ocrJobSpecFromServer.OffChainReportingSpec.UpdatedAt.String(), "20")
}

func runDirectRequestJobSpecAssertions(t *testing.T, ereJobSpecFromFile job.Job, ereJobSpecFromServer presenters.JobResource) {
	assert.Equal(t, ereJobSpecFromFile.DirectRequestSpec.ContractAddress, ereJobSpecFromServer.DirectRequestSpec.ContractAddress)
	assert.Equal(t, ereJobSpecFromFile.Pipeline.Source, ereJobSpecFromServer.PipelineSpec.DotDAGSource)
	// Check that create and update dates are non empty values.
	// Empty date value is "0001-01-01 00:00:00 +0000 UTC" so we are checking for the
	// millennia and century characters to be present
	assert.Contains(t, ereJobSpecFromServer.DirectRequestSpec.CreatedAt.String(), "20")
	assert.Contains(t, ereJobSpecFromServer.DirectRequestSpec.UpdatedAt.String(), "20")
}

func setupBridges(t *testing.T, db *sqlx.DB, cfg pg.QConfig) (b1, b2 string) {
	_, bridge := cltest.MustCreateBridge(t, db, cltest.BridgeOpts{}, cfg)
	_, bridge2 := cltest.MustCreateBridge(t, db, cltest.BridgeOpts{}, cfg)
	return bridge.Name.String(), bridge2.Name.String()
}

func setupJobsControllerTests(t *testing.T) (ta *cltest.TestApplication, cc cltest.HTTPClientCleaner) {
	cfg := configtest.NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {
		c.OCR.Enabled = ptr(true)
		c.P2P.V2.Enabled = ptr(true)
		c.P2P.V2.ListenAddresses = &[]string{fmt.Sprintf("127.0.0.1:%d", freeport.GetOne(t))}
		c.P2P.PeerID = &cltest.DefaultP2PPeerID
	})
	ec := setupEthClientForControllerTests(t)
	app := cltest.NewApplicationWithConfigAndKey(t, cfg, cltest.DefaultP2PKey, ec)
	require.NoError(t, app.Start(testutils.Context(t)))

	client := app.NewHTTPClient(nil)
	vrfKeyStore := app.GetKeyStore().VRF()
	_, err := vrfKeyStore.Create()
	require.NoError(t, err)
	return app, client
}

func setupEthClientForControllerTests(t *testing.T) *evmclimocks.Client {
	ec := cltest.NewEthMocksWithStartupAssertions(t)
	ec.On("PendingNonceAt", mock.Anything, mock.Anything).Return(uint64(0), nil).Maybe()
	ec.On("LatestBlockHeight", mock.Anything).Return(big.NewInt(100), nil).Maybe()
	ec.On("BalanceAt", mock.Anything, mock.Anything, mock.Anything).Once().Return(big.NewInt(0), nil).Maybe()
	return ec
}
