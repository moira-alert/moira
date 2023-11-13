package redis

import "time"

// DatabaseConfig - Redis database connection config
type DatabaseConfig struct {
	MasterName       string
	Addrs            []string
	Username         string
	Password         string
	SentinelPassword string
	SentinelUsername string
	MetricsTTL       time.Duration
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxRetries       int
}

type NotificationHistoryConfig struct {
	NotificationHistoryTTL        time.Duration
	NotificationHistoryQueryLimit int
}
