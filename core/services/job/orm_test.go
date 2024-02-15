package job_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jmoiron/sqlx"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/bridges"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/configtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/evmtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/logger"
	clnull "github.com/O1MaGnUmO1/erinaceus-vrf/core/null"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/keystore"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pg"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pipeline"
)

func NewTestORM(t *testing.T, db *sqlx.DB, pipelineORM pipeline.ORM, bridgeORM bridges.ORM, keyStore keystore.Master, cfg pg.QConfig) job.ORM {
	o := job.NewORM(db, pipelineORM, bridgeORM, keyStore, logger.TestLogger(t), cfg)
	t.Cleanup(func() { assert.NoError(t, o.Close()) })
	return o
}

func TestSetDRMinIncomingConfirmations(t *testing.T) {
	t.Parallel()

	config := configtest.NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {
		hundred := uint32(100)
		c.EVM[0].MinIncomingConfirmations = &hundred
	})
	chainConfig := evmtest.NewChainScopedConfig(t, config)

	jobSpec10 := job.DirectRequestSpec{
		MinIncomingConfirmations: clnull.Uint32From(10),
	}

	drs10 := job.SetDRMinIncomingConfirmations(chainConfig.EVM().MinIncomingConfirmations(), jobSpec10)
	assert.Equal(t, uint32(100), drs10.MinIncomingConfirmations.Uint32)

	jobSpec200 := job.DirectRequestSpec{
		MinIncomingConfirmations: clnull.Uint32From(200),
	}

	drs200 := job.SetDRMinIncomingConfirmations(chainConfig.EVM().MinIncomingConfirmations(), jobSpec200)
	assert.True(t, drs200.MinIncomingConfirmations.Valid)
	assert.Equal(t, uint32(200), drs200.MinIncomingConfirmations.Uint32)
}
