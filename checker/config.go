package checker

import "time"

type Config struct {
	Enabled              bool
	NoDataCheckInterval  time.Duration
	CheckInterval        int64
	MetricsTTL           int64
	StopCheckingInterval int64
	LogFile              string
	LogLevel             string
}
