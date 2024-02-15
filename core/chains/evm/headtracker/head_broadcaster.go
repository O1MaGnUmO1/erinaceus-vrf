package headtracker

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/logger"
	"github.com/O1MaGnUmO1/erinaceus-vrf/common/headtracker"
	commontypes "github.com/O1MaGnUmO1/erinaceus-vrf/common/types"
	evmtypes "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/types"
)

type headBroadcaster = headtracker.HeadBroadcaster[*evmtypes.Head, common.Hash]

var _ commontypes.HeadBroadcaster[*evmtypes.Head, common.Hash] = &headBroadcaster{}

func NewHeadBroadcaster(
	lggr logger.Logger,
) *headBroadcaster {
	return headtracker.NewHeadBroadcaster[*evmtypes.Head, common.Hash](lggr)
}
