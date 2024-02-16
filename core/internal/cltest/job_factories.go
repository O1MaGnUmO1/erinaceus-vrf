package cltest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/bridges"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/configtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/logger"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pipeline"
)

func MustInsertWebhookSpec(t *testing.T, db *sqlx.DB) (job.Job, job.WebhookSpec) {
	jobORM, pipelineORM := getORMs(t, db)
	webhookSpec := job.WebhookSpec{}
	require.NoError(t, jobORM.InsertWebhookSpec(&webhookSpec))

	pSpec := pipeline.Pipeline{}
	pipelineSpecID, err := pipelineORM.CreateSpec(pSpec, 0)
	require.NoError(t, err)

	createdJob := job.Job{WebhookSpecID: &webhookSpec.ID, WebhookSpec: &webhookSpec, SchemaVersion: 1, Type: "webhook",
		ExternalJobID: uuid.New(), PipelineSpecID: pipelineSpecID}
	require.NoError(t, jobORM.InsertJob(&createdJob))

	return createdJob, webhookSpec
}

func getORMs(t *testing.T, db *sqlx.DB) (jobORM job.ORM, pipelineORM pipeline.ORM) {
	config := configtest.NewTestGeneralConfig(t)
	keyStore := NewKeyStore(t, db, config.Database())
	lggr := logger.TestLogger(t)
	pipelineORM = pipeline.NewORM(db, lggr, config.Database(), config.JobPipeline().MaxSuccessfulRuns())
	bridgeORM := bridges.NewORM(db, lggr, config.Database())
	jobORM = job.NewORM(db, pipelineORM, bridgeORM, keyStore, lggr, config.Database())
	t.Cleanup(func() { jobORM.Close() })
	return
}
