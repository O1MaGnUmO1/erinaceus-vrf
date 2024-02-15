package resolver

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink/v2/core/bridges"
	"github.com/smartcontractkit/chainlink/v2/core/chains"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	evmrelay "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
	"github.com/smartcontractkit/chainlink/v2/core/utils/stringutils"
)

// Bridge retrieves a bridges by name.
func (r *Resolver) Bridge(ctx context.Context, args struct{ ID graphql.ID }) (*BridgePayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	name, err := bridges.ParseBridgeName(string(args.ID))
	if err != nil {
		return nil, err
	}

	bridge, err := r.App.BridgeORM().FindBridge(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewBridgePayload(bridge, err), nil
		}

		return nil, err
	}

	return NewBridgePayload(bridge, nil), nil
}

// Bridges retrieves a paginated list of bridges.
func (r *Resolver) Bridges(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*BridgesPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)

	brdgs, count, err := r.App.BridgeORM().BridgeTypes(offset, limit)
	if err != nil {
		return nil, err
	}

	return NewBridgesPayload(brdgs, int32(count)), nil
}

// Chain retrieves a chain by id.
func (r *Resolver) Chain(ctx context.Context, args struct{ ID graphql.ID }) (*ChainPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	cs, _, err := r.App.EVMORM().Chains(relay.ChainID(args.ID))
	if err != nil {
		return nil, err
	}
	l := len(cs)
	if l == 0 {
		return NewChainPayload(types.ChainStatus{}, chains.ErrNotFound), nil
	}
	if l > 1 {
		return nil, fmt.Errorf("multiple chains found: %d", len(cs))
	}
	return NewChainPayload(cs[0], nil), nil
}

// Chains retrieves a paginated list of chains.
func (r *Resolver) Chains(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*ChainsPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)

	chains, count, err := r.App.EVMORM().Chains()
	if err != nil {
		return nil, err
	}
	// bound the chain results
	if offset >= len(chains) {
		return nil, fmt.Errorf("offset %d out of range", offset)
	}
	end := len(chains)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return NewChainsPayload(chains[offset:end], int32(count)), nil
}

// Job retrieves a job by id.
func (r *Resolver) Job(ctx context.Context, args struct{ ID graphql.ID }) (*JobPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	id, err := stringutils.ToInt32(string(args.ID))
	if err != nil {
		return nil, err
	}

	j, err := r.App.JobORM().FindJobWithoutSpecErrors(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewJobPayload(r.App, nil, err), nil
		}

		//We still need to show the job in UI/CLI even if the chain id is disabled
		if errors.Is(err, chains.ErrNoSuchChainID) {
			return NewJobPayload(r.App, &j, err), nil
		}

		return nil, err
	}

	return NewJobPayload(r.App, &j, nil), nil
}

// Jobs fetches a paginated list of jobs
func (r *Resolver) Jobs(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*JobsPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)

	jobs, count, err := r.App.JobORM().FindJobs(offset, limit)
	if err != nil {
		return nil, err
	}

	return NewJobsPayload(r.App, jobs, int32(count)), nil
}

func (r *Resolver) CSAKeys(ctx context.Context) (*CSAKeysPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	keys, err := r.App.GetKeyStore().CSA().GetAll()
	if err != nil {
		return nil, err
	}

	return NewCSAKeysResolver(keys), nil
}

// Features retrieves each featured enabled by boolean mapping
func (r *Resolver) Features(ctx context.Context) (*FeaturesPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	return NewFeaturesPayloadResolver(r.App.GetConfig().Feature()), nil
}

// Node retrieves a node by ID (Name)
func (r *Resolver) Node(ctx context.Context, args struct{ ID graphql.ID }) (*NodePayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}
	r.App.GetLogger().Debug("resolver Node args %v", args)
	name := string(args.ID)
	r.App.GetLogger().Debug("resolver Node name %s", name)
	node, err := r.App.EVMORM().NodeStatus(name)
	if err != nil {
		r.App.GetLogger().Errorw("resolver getting node status", "err", err)

		if errors.Is(err, chains.ErrNotFound) {
			npr, warn := NewNodePayloadResolver(nil, err)
			if warn != nil {
				r.App.GetLogger().Warnw("Error creating NodePayloadResolver", "name", name, "err", warn)
			}
			return npr, nil
		}
		return nil, err
	}

	npr, warn := NewNodePayloadResolver(&node, nil)
	if warn != nil {
		r.App.GetLogger().Warnw("Error creating NodePayloadResolver", "name", name, "err", warn)
	}
	return npr, nil
}

func (r *Resolver) P2PKeys(ctx context.Context) (*P2PKeysPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	p2pKeys, err := r.App.GetKeyStore().P2P().GetAll()
	if err != nil {
		return nil, err
	}

	return NewP2PKeysPayload(p2pKeys), nil
}

// VRFKeys fetches all VRF keys.
func (r *Resolver) VRFKeys(ctx context.Context) (*VRFKeysPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	keys, err := r.App.GetKeyStore().VRF().GetAll()
	if err != nil {
		return nil, err
	}

	return NewVRFKeysPayloadResolver(keys), nil
}

// VRFKey fetches the VRF key with the given ID.
func (r *Resolver) VRFKey(ctx context.Context, args struct {
	ID graphql.ID
}) (*VRFKeyPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	key, err := r.App.GetKeyStore().VRF().Get(string(args.ID))
	if err != nil {
		if errors.Is(errors.Cause(err), keystore.ErrMissingVRFKey) {
			return NewVRFKeyPayloadResolver(vrfkey.KeyV2{}, err), nil
		}
		return nil, err
	}

	return NewVRFKeyPayloadResolver(key, nil), err
}

// Nodes retrieves a paginated list of nodes.
func (r *Resolver) Nodes(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*NodesPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)
	r.App.GetLogger().Debugw("resolver Nodes query", "offset", offset, "limit", limit)
	allNodes, total, err := r.App.GetRelayers().NodeStatuses(ctx, offset, limit)
	r.App.GetLogger().Debugw("resolver Nodes query result", "nodes", allNodes, "total", total, "err", err)

	if err != nil {
		r.App.GetLogger().Errorw("Error creating get nodes status from app", "err", err)
		return nil, err
	}
	npr, warn := NewNodesPayload(allNodes, int32(total))
	if warn != nil {
		r.App.GetLogger().Warnw("Error creating NodesPayloadResolver", "err", warn)
	}
	return npr, nil
}

func (r *Resolver) JobRuns(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*JobRunsPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	limit := pageLimit(args.Limit)
	offset := pageOffset(args.Offset)

	runs, count, err := r.App.JobORM().PipelineRuns(nil, offset, limit)
	if err != nil {
		return nil, err
	}

	return NewJobRunsPayload(runs, int32(count), r.App), nil
}

func (r *Resolver) JobRun(ctx context.Context, args struct {
	ID graphql.ID
}) (*JobRunPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	id, err := stringutils.ToInt64(string(args.ID))
	if err != nil {
		return nil, err
	}

	jr, err := r.App.JobORM().FindPipelineRunByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewJobRunPayload(nil, r.App, err), nil
		}

		return nil, err
	}

	return NewJobRunPayload(&jr, r.App, err), nil
}

func (r *Resolver) ETHKeys(ctx context.Context) (*ETHKeysPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	ks := r.App.GetKeyStore().Eth()

	keys, err := ks.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting unlocked keys: %v", err)
	}

	states, err := ks.GetStatesForKeys(keys)
	if err != nil {
		return nil, fmt.Errorf("error getting key states: %v", err)
	}

	var ethKeys []ETHKey

	for _, state := range states {
		k, err := ks.Get(state.Address.Hex())
		if err != nil {
			return nil, err
		}

		chain, err := r.App.GetRelayers().LegacyEVMChains().Get(state.EVMChainID.String())
		if errors.Is(errors.Cause(err), evmrelay.ErrNoChains) {
			ethKeys = append(ethKeys, ETHKey{
				addr:  k.EIP55Address,
				state: state,
			})

			continue
		}
		// Don't include keys without valid chain.
		// OperatorUI fails to show keys where chains are not in the config.
		if err == nil {
			ethKeys = append(ethKeys, ETHKey{
				addr:  k.EIP55Address,
				state: state,
				chain: chain,
			})
		}
	}
	// Put disabled keys to the end
	sort.SliceStable(ethKeys, func(i, j int) bool {
		return !states[i].Disabled && states[j].Disabled
	})

	return NewETHKeysPayload(ethKeys), nil
}

// ConfigV2 retrieves the Chainlink node's configuration (V2 mode)
func (r *Resolver) ConfigV2(ctx context.Context) (*ConfigV2PayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	cfg := r.App.GetConfig()
	return NewConfigV2Payload(cfg.ConfigTOML()), nil
}

func (r *Resolver) EthTransaction(ctx context.Context, args struct {
	Hash graphql.ID
}) (*EthTransactionPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	hash := common.HexToHash(string(args.Hash))
	etx, err := r.App.TxmStorageService().FindTxByHash(hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewEthTransactionPayload(nil, err), nil
		}

		return nil, err
	}

	return NewEthTransactionPayload(etx, err), nil
}

func (r *Resolver) EthTransactions(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*EthTransactionsPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)

	txs, count, err := r.App.TxmStorageService().Transactions(offset, limit)
	if err != nil {
		return nil, err
	}

	return NewEthTransactionsPayload(txs, int32(count)), nil
}

func (r *Resolver) EthTransactionsAttempts(ctx context.Context, args struct {
	Offset *int32
	Limit  *int32
}) (*EthTransactionsAttemptsPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	offset := pageOffset(args.Offset)
	limit := pageLimit(args.Limit)

	attempts, count, err := r.App.TxmStorageService().TxAttempts(offset, limit)
	if err != nil {
		return nil, err
	}

	return NewEthTransactionsAttemptsPayload(attempts, int32(count)), nil
}

func (r *Resolver) GlobalLogLevel(ctx context.Context) (*GlobalLogLevelPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	logLevel := r.App.GetConfig().Log().Level().String()

	return NewGlobalLogLevelPayload(logLevel), nil
}

func (r *Resolver) SQLLogging(ctx context.Context) (*GetSQLLoggingPayloadResolver, error) {
	if err := authenticateUser(ctx); err != nil {
		return nil, err
	}

	enabled := r.App.GetConfig().Database().LogSQL()

	return NewGetSQLLoggingPayload(enabled), nil
}