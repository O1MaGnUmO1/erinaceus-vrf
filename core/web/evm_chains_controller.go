package web

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/relay"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/presenters"
)

var ErrEVMNotEnabled = errChainDisabled{name: "EVM", tomlKey: "EVM.Enabled"}

func NewEVMChainsController(app erinaceus.Application) ChainsController {
	return newChainsController[presenters.EVMChainResource](
		relay.EVM,
		app.GetRelayers().List(erinaceus.FilterRelayersByType(relay.EVM)),
		ErrEVMNotEnabled,
		presenters.NewEVMChainResource,
		app.GetLogger(),
		app.GetAuditLogger())
}
