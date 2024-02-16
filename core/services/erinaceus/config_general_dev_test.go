//go:build dev

package erinaceus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/logger"
)

// Tests in this file only run in dev mode
// /usr/bin/go test --tags=dev -timeout 360s -run ^TestTOMLGeneralConfig_DevModeInsecureConfig github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus

func TestTOMLGeneralConfig_DevModeInsecureConfig(t *testing.T) {
	t.Parallel()

	t.Run("all insecure configs are false by default", func(t *testing.T) {
		config, err := GeneralConfigOpts{}.New(logger.TestLogger(t))
		require.NoError(t, err)

		assert.False(t, config.Insecure().DevWebServer())
		assert.False(t, config.Insecure().DisableRateLimiting())
		assert.False(t, config.Insecure().InfiniteDepthQueries())
	})

	t.Run("insecure config ignore override on non-dev builds", func(t *testing.T) {
		config, err := GeneralConfigOpts{
			OverrideFn: func(c *Config, s *Secrets) {
				*c.Insecure.DevWebServer = true
				*c.Insecure.DisableRateLimiting = true
				*c.Insecure.InfiniteDepthQueries = true
			}}.New(logger.TestLogger(t))
		require.NoError(t, err)

		assert.True(t, config.Insecure().DevWebServer())
		assert.True(t, config.Insecure().DisableRateLimiting())
		assert.True(t, config.Insecure().InfiniteDepthQueries())
	})

	t.Run("ParseConfig accepts insecure values on dev builds", func(t *testing.T) {
		opts := GeneralConfigOpts{}
		err := opts.ParseConfig(`
		  [insecure]
		  DevWebServer = true
		`)
		cfg, err := opts.init()
		require.NoError(t, err)
		err = cfg.c.Validate()
		require.NoError(t, err)
	})
}
