package web

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/keystore/keys/dkgencryptkey"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/presenters"
)

func NewDKGEncryptKeysController(app chainlink.Application) KeysController {
	return NewKeysController[dkgencryptkey.Key, presenters.DKGEncryptKeyResource](
		app.GetKeyStore().DKGEncrypt(),
		app.GetLogger(),
		app.GetAuditLogger(),
		"dkgencryptKey",
		presenters.NewDKGEncryptKeyResource,
		presenters.NewDKGEncryptKeyResources)
}
