package chainlink

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
)

type prometheusConfig struct {
	s toml.PrometheusSecrets
}

func (p *prometheusConfig) AuthToken() string {
	if p.s.AuthToken == nil {
		return ""
	}
	return string(*p.s.AuthToken)
}
