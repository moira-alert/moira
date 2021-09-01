package redis

import "time"

// Config - Redis database connection config
type Config struct {
	MasterName string
	Addrs      []string
	Username   string
	Password   string
	MetricsTTL time.Duration
}
