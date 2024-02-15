package chainlink

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
)

type sentryConfig struct {
	c toml.Sentry
}

func (s sentryConfig) DSN() string {
	return *s.c.DSN
}

func (s sentryConfig) Debug() bool {
	return *s.c.Debug
}

func (s sentryConfig) Environment() string {
	return *s.c.Environment
}

func (s sentryConfig) Release() string {
	return *s.c.Release
}
