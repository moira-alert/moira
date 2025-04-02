package redis

import "time"

// DatabaseConfig - Redis database connection config.
type DatabaseConfig struct {
	Addrs []string

	Username string
	Password string

	MasterName       string
	SentinelPassword string
	SentinelUsername string

	MetricsTTL   time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	MaxRedirects    int
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration

	PoolTimeout time.Duration
	PoolSize    int

	ReadOnly       bool
	RouteByLatency bool
	RouteRandomly  bool
}

type NotificationHistoryConfig struct {
	NotificationHistoryTTL time.Duration
}

// Notifier configuration in redis.
type NotificationConfig struct {
	// Need to determine if notification is delayed - the difference between creation time and sending time
	// is greater than DelayedTime
	DelayedTime time.Duration
	// TransactionTimeout defines the timeout between fetch notifications transactions
	TransactionTimeout time.Duration
	// TransactionMaxRetries defines the maximum number of attempts to make a transaction
	TransactionMaxRetries int
	// TransactionHeuristicLimit maximum allowable limit, after this limit all notifications
	// without limit will be taken
	TransactionHeuristicLimit int64
	// ResaveTime is the time by which the timestamp of notifications with triggers
	// or metrics on Maintenance is incremented
	ResaveTime time.Duration
}
