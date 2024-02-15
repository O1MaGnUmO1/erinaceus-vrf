package erinaceus

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config"
)

//go:generate mockery --quiet --name GeneralConfig --output ./mocks/ --case=underscore

type GeneralConfig interface {
	config.AppConfig
	toml.HasEVMConfigs
	// ConfigTOML returns both the user provided and effective configuration as TOML.
	ConfigTOML() (user, effective string)
}
