package redis

import "time"

// Config - Redis database connection config
type Config struct {
	MasterName       string
	Addrs            []string
	Username         string
	Password         string
	SentinelPassword string
	MetricsTTL       time.Duration
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxRetries       int
}
