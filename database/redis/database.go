package redis

import (
	"context"
	"time"

	"github.com/moira-alert/moira/clock"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
)

const metricEventChannelSize = 16384

const (
	cacheCleanupInterval         = time.Minute * 60
	cacheValueExpirationDuration = time.Minute
)

// DBSource is type for describing who create database instance
type DBSource string

// All types of database users
const (
	API        DBSource = "API"
	Checker    DBSource = "Checker"
	Filter     DBSource = "Filter"
	Notifier   DBSource = "Notifier"
	Cli        DBSource = "Cli"
	testSource DBSource = "test"
)

// DbConnector contains redis client
type DbConnector struct {
	client               *redis.UniversalClient
	logger               moira.Logger
	retentionCache       *cache.Cache
	retentionSavingCache *cache.Cache
	metricsCache         *cache.Cache
	sync                 *redsync.Redsync
	metricsTTLSeconds    int64
	context              context.Context
	source               DBSource
	clock                moira.Clock
	notificationHistory  NotificationHistoryConfig
	// Notifier configuration in redis
	notification NotificationConfig
}

func NewDatabase(logger moira.Logger, config DatabaseConfig, nh NotificationHistoryConfig, n NotificationConfig, source DBSource) *DbConnector {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		MasterName:       config.MasterName,
		Addrs:            config.Addrs,
		Username:         config.Username,
		Password:         config.Password,
		SentinelPassword: config.SentinelPassword,
		SentinelUsername: config.SentinelUsername,
		DialTimeout:      config.DialTimeout,
		ReadTimeout:      config.ReadTimeout,
		WriteTimeout:     config.WriteTimeout,
		MaxRetries:       config.MaxRetries,
	})

	ctx := context.Background()

	syncPool := goredis.NewPool(client)

	connector := DbConnector{
		client:               &client,
		logger:               logger,
		context:              ctx,
		retentionCache:       cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		retentionSavingCache: cache.New(cache.NoExpiration, cache.DefaultExpiration),
		metricsCache:         cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		sync:                 redsync.New(syncPool),
		metricsTTLSeconds:    int64(config.MetricsTTL.Seconds()),
		source:               source,
		clock:                clock.NewSystemClock(),
		notificationHistory:  nh,
		notification:         n,
	}

	return &connector
}

// NewTestDatabase use it only for tests
func NewTestDatabase(logger moira.Logger) *DbConnector {
	return NewDatabase(logger, DatabaseConfig{
		Addrs: []string{"0.0.0.0:6379"},
	},
		NotificationHistoryConfig{
			NotificationHistoryTTL:        time.Hour * 48,
			NotificationHistoryQueryLimit: 1000,
		},
		NotificationConfig{
			DelayedTime:               time.Minute,
			TransactionTimeout:        200 * time.Millisecond,
			TransactionMaxRetries:     10,
			TransactionHeuristicLimit: 10000,
		},
		testSource)
}

// NewTestDatabaseWithIncorrectConfig use it only for tests
func NewTestDatabaseWithIncorrectConfig(logger moira.Logger) *DbConnector {
	return NewDatabase(logger,
		DatabaseConfig{Addrs: []string{"0.0.0.0:0000"}},
		NotificationHistoryConfig{
			NotificationHistoryTTL:        time.Hour * 48,
			NotificationHistoryQueryLimit: 1000,
		},
		NotificationConfig{
			DelayedTime:               time.Minute,
			TransactionTimeout:        200 * time.Millisecond,
			TransactionMaxRetries:     10,
			TransactionHeuristicLimit: 10000,
		},
		testSource)
}

// Flush deletes all the keys of the DB, use it only for tests
func (connector *DbConnector) Flush() {
	err := connector.callFunc(func(connector *DbConnector, client redis.UniversalClient) error {
		return client.FlushDB(connector.context).Err()
	})
	if err != nil {
		return
	}
}

// Get key ttl, use it only for tests
func (connector *DbConnector) getTTL(key string) time.Duration {
	return (*connector.client).PTTL(connector.context, key).Val()
}

// Delete the key, use it only for tests
func (connector *DbConnector) delete(key string) {
	(*connector.client).Del(connector.context, key)
}

func (connector *DbConnector) Client() redis.UniversalClient {
	return *connector.client
}

func (connector *DbConnector) Context() context.Context {
	return connector.context
}

// callFunc calls the fn dependent of Redis client type (cluster or standalone).
func (connector *DbConnector) callFunc(fn func(connector *DbConnector, client redis.UniversalClient) error) error {
	client := *connector.client

	switch c := client.(type) {
	case *redis.ClusterClient:
		return c.ForEachMaster(connector.context, func(ctx context.Context, shard *redis.Client) error {
			return fn(connector, shard)
		})
	default:
		return fn(connector, client)
	}
}
