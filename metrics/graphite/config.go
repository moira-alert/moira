package graphite

import "time"

// Config for graphite connection settings
type Config struct {
	Enabled      bool
	RuntimeStats bool
	URI          string
	Prefix       string
	Interval     time.Duration
}
