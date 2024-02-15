package config

import (
	ocrcommontypes "github.com/smartcontractkit/libocr/commontypes"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

type V2 interface {
	Enabled() bool
	AnnounceAddresses() []string
	DefaultBootstrappers() (locators []ocrcommontypes.BootstrapperLocator)
	DeltaDial() models.Duration
	DeltaReconcile() models.Duration
	ListenAddresses() []string
}
