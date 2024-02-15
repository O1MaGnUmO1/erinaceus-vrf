package config

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

type AuditLogger interface {
	Enabled() bool
	ForwardToUrl() (models.URL, error)
	Environment() string
	JsonWrapperKey() string
	Headers() (models.ServiceHeaders, error)
}
