package config

import (
	"time"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

type JobPipeline interface {
	DefaultHTTPLimit() int64
	DefaultHTTPTimeout() models.Duration
	MaxRunDuration() time.Duration
	MaxSuccessfulRuns() uint64
	ReaperInterval() time.Duration
	ReaperThreshold() time.Duration
	ResultWriteQueueDepth() uint64
	ExternalInitiatorsEnabled() bool
}
