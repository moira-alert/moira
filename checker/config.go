package checker

import "time"

type Config struct {
	NoDataCheckInterval  time.Duration
	CheckInterval        int64
	MetricsTTL           int64
	StopCheckingInterval int64
}
