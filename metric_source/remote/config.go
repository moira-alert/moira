package remote

import (
	"github.com/moira-alert/moira/metric_source/retries"
	"time"
)

// Config represents config from remote storage.
type Config struct {
	URL                     string
	CheckInterval           time.Duration
	MetricsTTL              time.Duration
	Timeout                 time.Duration
	User                    string
	Password                string
	RetrySeconds            []time.Duration
	HealthCheckTimeout      time.Duration
	HealthCheckRetrySeconds []time.Duration
	Retries                 retries.Config
	HealthcheckRetries      *retries.Config
}
