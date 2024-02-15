package docs

import (
	"log"
	"strings"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/config"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink/cfgtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/dialects"
)

var (
	defaults toml.Core
)

func init() {
	if err := cfgtest.DocDefaultsOnly(strings.NewReader(coreTOML), &defaults, config.DecodeTOML); err != nil {
		log.Fatalf("Failed to initialize defaults from docs: %v", err)
	}
}

func CoreDefaults() (c toml.Core) {
	c.SetFrom(&defaults)
	c.Database.Dialect = dialects.Postgres // not user visible - overridden for tests only
	c.Tracing.Attributes = make(map[string]string)
	return
}
