package web

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/relay"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/presenters"
)

func NewEVMNodesController(app chainlink.Application) NodesController {
	scopedNodeStatuser := NewNetworkScopedNodeStatuser(app.GetRelayers(), relay.EVM)

	return newNodesController[presenters.EVMNodeResource](
		scopedNodeStatuser, ErrEVMNotEnabled, presenters.NewEVMNodeResource, app.GetAuditLogger())
}
