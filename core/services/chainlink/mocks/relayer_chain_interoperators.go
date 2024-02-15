package mocks

import (
	"context"
	"slices"

	services2 "github.com/O1MaGnUmO1/erinaceus-vrf/core/services"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/legacyevm"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/relay"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/types"
)

// FakeRelayerChainInteroperators is a fake chainlink.RelayerChainInteroperators.
// This exists because mockery generation doesn't understand how to produce an alias instead of the underlying type (which is not exported in this case).
type FakeRelayerChainInteroperators struct {
	EVMChains legacyevm.LegacyChainContainer
	Nodes     []types.NodeStatus
	NodesErr  error
}

func (f *FakeRelayerChainInteroperators) LegacyEVMChains() legacyevm.LegacyChainContainer {
	return f.EVMChains
}

func (f *FakeRelayerChainInteroperators) NodeStatuses(ctx context.Context, offset, limit int, relayIDs ...relay.ID) (nodes []types.NodeStatus, count int, err error) {
	return slices.Clone(f.Nodes), len(f.Nodes), f.NodesErr
}

func (f *FakeRelayerChainInteroperators) Services() []services2.ServiceCtx {
	panic("unimplemented")
}

func (f *FakeRelayerChainInteroperators) List(filter chainlink.FilterFn) chainlink.RelayerChainInteroperators {
	panic("unimplemented")
}

func (f *FakeRelayerChainInteroperators) Get(id relay.ID) (loop.Relayer, error) {
	panic("unimplemented")
}

func (f *FakeRelayerChainInteroperators) Slice() []loop.Relayer {
	panic("unimplemented")
}

func (f *FakeRelayerChainInteroperators) ChainStatus(ctx context.Context, id relay.ID) (types.ChainStatus, error) {
	panic("unimplemented")
}

func (f *FakeRelayerChainInteroperators) ChainStatuses(ctx context.Context, offset, limit int) ([]types.ChainStatus, int, error) {
	panic("unimplemented")
}
