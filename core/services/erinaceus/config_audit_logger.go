package erinaceus

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/build"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

type auditLoggerConfig struct {
	c toml.AuditLogger
}

func (a auditLoggerConfig) Enabled() bool {
	return *a.c.Enabled
}

func (a auditLoggerConfig) ForwardToUrl() (models.URL, error) {
	return *a.c.ForwardToUrl, nil
}

func (a auditLoggerConfig) Environment() string {
	if !build.IsProd() {
		return "develop"
	}
	return "production"
}

func (a auditLoggerConfig) JsonWrapperKey() string {
	return *a.c.JsonWrapperKey
}

func (a auditLoggerConfig) Headers() (models.ServiceHeaders, error) {
	return *a.c.Headers, nil
}
