package checker

import "time"

// Config represent checker config
type Config struct {
	Enabled                     bool
	NoDataCheckInterval         time.Duration
	CheckInterval               time.Duration
	MetricsTTLSeconds           int64
	StopCheckingIntervalSeconds int64
	MaxParallelChecks           int
	LogFile                     string
	LogLevel                    string
}
