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
	MetricsTTLSeconds           int64
	StopCheckingIntervalSeconds int64
	MaxParallelChecks           int
	MaxParallelRemoteChecks     int
	LogFile                     string
	LogLevel                    string
}
