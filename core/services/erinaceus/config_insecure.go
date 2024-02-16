package erinaceus

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/build"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
)

type insecureConfig struct {
	c toml.Insecure
}

func (i *insecureConfig) DevWebServer() bool {
	return build.IsDev() && i.c.DevWebServer != nil &&
		*i.c.DevWebServer
}

func (i *insecureConfig) DisableRateLimiting() bool {
	return build.IsDev() && i.c.DisableRateLimiting != nil &&
		*i.c.DisableRateLimiting
}

func (i *insecureConfig) InfiniteDepthQueries() bool {
	return build.IsDev() && i.c.InfiniteDepthQueries != nil &&
		*i.c.InfiniteDepthQueries
}
