package remote

import "time"

// Config represents config from remote storage.
type Config struct {
	URL           string
	CheckInterval time.Duration
	MetricsTTL    time.Duration
	Timeout       time.Duration
	User          string
	Password      string
}
