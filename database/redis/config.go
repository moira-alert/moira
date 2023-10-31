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

// Notifier configuration in redis
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
	// ResaveTime is the time to resave a notification whose trigger or metric is in Maintenance,
	// for more details see PR #946
	ResaveTime time.Duration
}
