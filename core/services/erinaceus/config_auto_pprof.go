package erinaceus

import (
	"path/filepath"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/utils"
)

var _ config.AutoPprof = (*autoPprofConfig)(nil)

type autoPprofConfig struct {
	c       toml.AutoPprof
	rootDir func() string
}

func (a *autoPprofConfig) Enabled() bool {
	return *a.c.Enabled
}

func (a *autoPprofConfig) BlockProfileRate() int {
	return int(*a.c.BlockProfileRate)
}

func (a *autoPprofConfig) CPUProfileRate() int {
	return int(*a.c.CPUProfileRate)
}

func (a *autoPprofConfig) GatherDuration() models.Duration {
	return models.MustMakeDuration(a.c.GatherDuration.Duration())
}

func (a *autoPprofConfig) GatherTraceDuration() models.Duration {
	return models.MustMakeDuration(a.c.GatherTraceDuration.Duration())
}

func (a *autoPprofConfig) GoroutineThreshold() int {
	return int(*a.c.GoroutineThreshold)
}

func (a *autoPprofConfig) MaxProfileSize() utils.FileSize {
	return *a.c.MaxProfileSize
}

func (a *autoPprofConfig) MemProfileRate() int {
	return int(*a.c.MemProfileRate)
}

func (a *autoPprofConfig) MemThreshold() utils.FileSize {
	return *a.c.MemThreshold
}

func (a *autoPprofConfig) MutexProfileFraction() int {
	return int(*a.c.MutexProfileFraction)
}

func (a *autoPprofConfig) PollInterval() models.Duration {
	return *a.c.PollInterval
}

func (a *autoPprofConfig) ProfileRoot() string {
	s := *a.c.ProfileRoot
	if s == "" {
		s = filepath.Join(a.rootDir(), "pprof")
	}
	return s
}
