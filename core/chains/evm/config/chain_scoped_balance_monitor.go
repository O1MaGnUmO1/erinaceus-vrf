package config

import "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/config/toml"

type balanceMonitorConfig struct {
	c toml.BalanceMonitor
}

func (b *balanceMonitorConfig) Enabled() bool {
	return *b.c.Enabled
}
