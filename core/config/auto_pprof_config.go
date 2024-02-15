package config

import (
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/utils"
)

type AutoPprof interface {
	BlockProfileRate() int
	CPUProfileRate() int
	Enabled() bool
	GatherDuration() models.Duration
	GatherTraceDuration() models.Duration
	GoroutineThreshold() int
	MaxProfileSize() utils.FileSize
	MemProfileRate() int
	MemThreshold() utils.FileSize
	MutexProfileFraction() int
	PollInterval() models.Duration
	ProfileRoot() string
}
