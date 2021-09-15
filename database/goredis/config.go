package goredis

import "time"

// Config - Redis database connection config
type Config struct {
	MasterName string
	Addrs      []string
	MetricsTTL time.Duration
}
