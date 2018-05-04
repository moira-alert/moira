package checker

import (
	"time"

	"github.com/moira-alert/moira/remote"
)

// Config represent checker config
type Config struct {
	Enabled                     bool
	NoDataCheckInterval         time.Duration
	CheckInterval               time.Duration
	MetricsTTLSeconds           int64
	StopCheckingIntervalSeconds int64
	MaxParallelChecks           int
	MaxParallelRemoteChecks     int
	LogFile                     string
	LogLevel                    string
	Remote                      remote.Config
}
