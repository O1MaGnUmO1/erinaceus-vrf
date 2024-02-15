package loader

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	commontypes "github.com/O1MaGnUmO1/chainlink-common/pkg/types"

	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/txmgr"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/types"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/v2/core/utils/stringutils"
)

// ErrInvalidType indicates that results loaded is not the type expected
var ErrInvalidType = errors.New("invalid type")

// GetChainByID fetches the chain by it's id.
func GetChainByID(ctx context.Context, id string) (*commontypes.ChainStatus, error) {
	ldr := For(ctx)

	thunk := ldr.ChainsByIDLoader.Load(ctx, dataloader.StringKey(id))
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	chain, ok := result.(commontypes.ChainStatus)
	if !ok {
		return nil, ErrInvalidType
	}

	return &chain, nil
}

// GetNodesByChainID fetches the nodes for a chain.
func GetNodesByChainID(ctx context.Context, id string) ([]types.Node, error) {
	ldr := For(ctx)

	thunk := ldr.NodesByChainIDLoader.Load(ctx, dataloader.StringKey(id))
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	nodes, ok := result.([]types.Node)
	if !ok {
		return nil, ErrInvalidType
	}

	return nodes, nil
}

// GetJobRunsByIDs fetches the job runs by their ID.
func GetJobRunsByIDs(ctx context.Context, ids []int64) ([]pipeline.Run, error) {
	ldr := For(ctx)

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = stringutils.FromInt64(id)
	}

	thunk := ldr.JobRunsByIDLoader.LoadMany(ctx, dataloader.NewKeysFromStrings(strIDs))
	results, errs := thunk()
	if errs != nil {
		merr := multierr.Combine(errs...)

		return nil, errors.Wrap(merr, "errors fetching runs")
	}

	runs := []pipeline.Run{}
	for _, result := range results {
		if run, ok := result.(pipeline.Run); ok {
			runs = append(runs, run)
		}
	}

	return runs, nil
}

// GetJobByExternalJobID fetches the job proposals by external job ID
func GetJobByExternalJobID(ctx context.Context, id string) (*job.Job, error) {
	ldr := For(ctx)

	thunk := ldr.JobsByExternalJobIDs.Load(ctx, dataloader.StringKey(id))
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	job, ok := result.(job.Job)
	if !ok {
		return nil, ErrInvalidType
	}

	return &job, nil
}

// GetJobByPipelineSpecID fetches the job by pipeline spec ID.
func GetJobByPipelineSpecID(ctx context.Context, id string) (*job.Job, error) {
	ldr := For(ctx)

	thunk := ldr.JobsByPipelineSpecIDLoader.Load(ctx, dataloader.StringKey(id))
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	jb, ok := result.(job.Job)
	if !ok {
		return nil, ErrInvalidType
	}

	return &jb, nil
}

// GetEthTxAttemptsByEthTxID fetches the attempts for an eth transaction.
func GetEthTxAttemptsByEthTxID(ctx context.Context, id string) ([]txmgr.TxAttempt, error) {
	ldr := For(ctx)

	thunk := ldr.EthTxAttemptsByEthTxIDLoader.Load(ctx, dataloader.StringKey(id))
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	attempts, ok := result.([]txmgr.TxAttempt)
	if !ok {
		return nil, ErrInvalidType
	}

	return attempts, nil
}

// GetJobSpecErrorsByJobID fetches the Spec Errors for a Job.
func GetJobSpecErrorsByJobID(ctx context.Context, jobID int32) ([]job.SpecError, error) {
	ldr := For(ctx)

	thunk := ldr.SpecErrorsByJobIDLoader.Load(ctx,
		dataloader.StringKey(stringutils.FromInt32(jobID)),
	)
	result, err := thunk()
	if err != nil {
		return nil, err
	}

	specErrs, ok := result.([]job.SpecError)
	if !ok {
		return nil, ErrInvalidType
	}

	return specErrs, nil
}
