package graphite

import "time"

//Config for graphite connection settings
type Config struct {
	URI      string
	Prefix   string
	Interval time.Duration
}
