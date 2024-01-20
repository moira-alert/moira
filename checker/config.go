package checker

import (
	"time"

	"github.com/moira-alert/moira"
)

// Config represent checker config
type Config struct {
	Enabled                     bool
	NoDataCheckInterval         time.Duration
	LazyTriggersCheckInterval   time.Duration
	SourceCheckConfigs          map[moira.ClusterKey]SourceCheckConfig
	StopCheckingIntervalSeconds int64
	LogFile                     string
	LogLevel                    string
	LogTriggersToLevel          map[string]string
	MetricEventPopBatchSize     int64
	MetricEventPopDelay         time.Duration
}

type SourceCheckConfig struct {
	Enabled           bool
	CheckInterval     time.Duration
	MaxParallelChecks int
}
