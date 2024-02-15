package job_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"
	"github.com/O1MaGnUmO1/chainlink-common/pkg/services"
	"github.com/O1MaGnUmO1/chainlink-common/pkg/utils"

	"github.com/smartcontractkit/chainlink/v2/core/bridges"
	evmtypes "github.com/smartcontractkit/chainlink/v2/core/chains/evm/types"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/configtest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	evmrelay "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
	evmrelayer "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
)

type delegate struct {
	jobType                    job.Type
	services                   []job.ServiceCtx
	jobID                      int32
	chContinueCreatingServices chan struct{}
	job.Delegate
}

func (d delegate) JobType() job.Type {
	return d.jobType
}

// ServicesForSpec satisfies the job.Delegate interface.
func (d delegate) ServicesForSpec(js job.Job) ([]job.ServiceCtx, error) {
	if js.Type != d.jobType {
		return nil, nil
	}
	return d.services, nil
}

func clearDB(t *testing.T, db *sqlx.DB) {
	cltest.ClearDBTables(t, db, "jobs", "pipeline_runs", "pipeline_specs", "pipeline_task_runs")
}

type relayGetter struct {
	e evmrelay.EVMChainRelayerExtender
	r *evmrelayer.Relayer
}

func (g *relayGetter) Get(id relay.ID) (loop.Relayer, error) {
	return evmrelayer.NewLoopRelayServerAdapter(g.r, g.e), nil
}

func TestSpawner_CreateJobDeleteJob(t *testing.T) {
	t.Parallel()

	config := configtest.NewTestGeneralConfig(t)
	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, config.Database())
	ethKeyStore := keyStore.Eth()
	require.NoError(t, keyStore.P2P().Add(cltest.DefaultP2PKey))

	_, _ = cltest.MustInsertRandomKey(t, ethKeyStore)

	ethClient := cltest.NewEthMocksWithDefaultChain(t)
	ethClient.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.Anything, false).
		Run(func(args mock.Arguments) {
			head := args.Get(1).(**evmtypes.Head)
			*head = cltest.Head(10)
		}).
		Return(nil).Maybe()

	t.Run("should respect its dependents", func(t *testing.T) {
		lggr := logger.TestLogger(t)
		orm := NewTestORM(t, db, pipeline.NewORM(db, lggr, config.Database(), config.JobPipeline().MaxSuccessfulRuns()), bridges.NewORM(db, lggr, config.Database()), keyStore, config.Database())
		a := utils.NewDependentAwaiter()
		a.AddDependents(1)
		spawner := job.NewSpawner(orm, config.Database(), noopChecker{}, map[job.Type]job.Delegate{}, db, lggr, []utils.DependentAwaiter{a})
		// Starting the spawner should signal to the dependents
		result := make(chan bool)
		go func() {
			select {
			case <-a.AwaitDependents():
				result <- true
			case <-time.After(2 * time.Second):
				result <- false
			}
		}()
		require.NoError(t, spawner.Start(testutils.Context(t)))
		assert.True(t, <-result, "failed to signal to dependents")
	})

	clearDB(t, db)

	clearDB(t, db)

}

type noopChecker struct{}

func (n noopChecker) Register(service services.HealthReporter) error { return nil }

func (n noopChecker) Unregister(name string) error { return nil }

func (n noopChecker) IsReady() (ready bool, errors map[string]error) { return true, nil }

func (n noopChecker) IsHealthy() (healthy bool, errors map[string]error) { return true, nil }

func (n noopChecker) Start() error { return nil }

func (n noopChecker) Close() error { return nil }
