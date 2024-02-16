package loader

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/graph-gophers/dataloader"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
)

type loadersKey struct{}

type Dataloader struct {
	app erinaceus.Application

	ChainsByIDLoader                *dataloader.Loader
	EthTxAttemptsByEthTxIDLoader    *dataloader.Loader
	JobProposalsByManagerIDLoader   *dataloader.Loader
	JobProposalSpecsByJobProposalID *dataloader.Loader
	JobRunsByIDLoader               *dataloader.Loader
	JobsByExternalJobIDs            *dataloader.Loader
	JobsByPipelineSpecIDLoader      *dataloader.Loader
	NodesByChainIDLoader            *dataloader.Loader
	SpecErrorsByJobIDLoader         *dataloader.Loader
}

func New(app erinaceus.Application) *Dataloader {
	var (
		nodes    = &nodeBatcher{app: app}
		chains   = &chainBatcher{app: app}
		jobRuns  = &jobRunBatcher{app: app}
		jbs      = &jobBatcher{app: app}
		attmpts  = &ethTransactionAttemptBatcher{app: app}
		specErrs = &jobSpecErrorsBatcher{app: app}
	)

	return &Dataloader{
		app: app,

		ChainsByIDLoader:             dataloader.NewBatchedLoader(chains.loadByIDs),
		EthTxAttemptsByEthTxIDLoader: dataloader.NewBatchedLoader(attmpts.loadByEthTransactionIDs),
		JobRunsByIDLoader:            dataloader.NewBatchedLoader(jobRuns.loadByIDs),
		JobsByExternalJobIDs:         dataloader.NewBatchedLoader(jbs.loadByExternalJobIDs),
		JobsByPipelineSpecIDLoader:   dataloader.NewBatchedLoader(jbs.loadByPipelineSpecIDs),
		NodesByChainIDLoader:         dataloader.NewBatchedLoader(nodes.loadByChainIDs),
		SpecErrorsByJobIDLoader:      dataloader.NewBatchedLoader(specErrs.loadByJobIDs),
	}
}

// Middleware injects the dataloader into a gin context.
func Middleware(app erinaceus.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := InjectDataloader(c.Request.Context(), app)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// InjectDataloader injects the dataloader into the context.
func InjectDataloader(ctx context.Context, app erinaceus.Application) context.Context {
	return context.WithValue(ctx, loadersKey{}, New(app))
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Dataloader {
	return ctx.Value(loadersKey{}).(*Dataloader)
}
