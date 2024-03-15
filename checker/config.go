package checker

import (
	"time"

	"github.com/moira-alert/moira"
)

// Config represent checker config.
type Config struct {
	Enabled                         bool
	LazyTriggersCheckInterval       time.Duration
	SourceCheckConfigs              map[moira.ClusterKey]SourceCheckConfig
	StopCheckingIntervalSeconds     int64
	LogFile                         string
	LogLevel                        string
	LogTriggersToLevel              map[string]string
	MetricEventPopBatchSize         int64
	MetricEventPopDelay             time.Duration
	CriticalTimeOfCheck             time.Duration
	MetricEventTriggerCheckInterval time.Duration
}

// SourceCheckConfig represents check parameters for a single metric source.
type SourceCheckConfig struct {
	CheckInterval     time.Duration
	MaxParallelChecks int
}
