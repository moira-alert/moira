package remote

import "time"

// Config represents config from remote storage
type Config struct {
	URL           string
	CheckInterval time.Duration
	MetricsTTL    time.Duration
	Timeout       time.Duration
	User          string
	Password      string
	Enabled       bool
}

// isEnabled checks that remote config is enabled (url is defined and enabled flag is set)
func (c *Config) isEnabled() bool {
	return c.Enabled && c.URL != ""
}
