package chainlink

import (
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/config/toml"
	"github.com/smartcontractkit/chainlink/v2/core/config"
)

//go:generate mockery --quiet --name GeneralConfig --output ./mocks/ --case=underscore

type GeneralConfig interface {
	config.AppConfig
	toml.HasEVMConfigs
	// ConfigTOML returns both the user provided and effective configuration as TOML.
	ConfigTOML() (user, effective string)
}
