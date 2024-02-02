package checker

import (
	"time"
)

// Config represent checker config
type Config struct {
	Enabled                     bool
	NoDataCheckInterval         time.Duration
	CheckInterval               time.Duration
	LazyTriggersCheckInterval   time.Duration
	StopCheckingIntervalSeconds int64
	MaxParallelLocalChecks      int
	MaxParallelRemoteChecks     int
	MaxParallelPrometheusChecks int
	LogFile                     string
	LogLevel                    string
	LogTriggersToLevel          map[string]string
	MetricEventPopBatchSize     int64
	MetricEventPopDelay         time.Duration
	CriticalTimeOfCheck         time.Duration
}
