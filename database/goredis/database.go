package goredis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
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
	client  *redis.UniversalClient
	logger  moira.Logger
	retentionSavingCache *cache.Cache
	context context.Context
	sync    *redsync.Redsync
	source  DBSource
}

func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: config.Addrs,
	})

	ctx := context.Background()

	syncPool := goredis.NewPool(client)

	connector := DbConnector{
		client:  &client,
		logger:  logger,
		context: ctx,
		retentionSavingCache: cache.New(cache.NoExpiration, cache.DefaultExpiration),
		source:  source,
		sync:    redsync.New(syncPool),
	}
	return &connector
}

// Deletes all the keys of the DB, use it only for tests
func (connector *DbConnector) flush() {
	(*connector.client).FlushDB(connector.context)
}

// Get key ttl, use it only for tests
func (connector *DbConnector) getTTL(key string) time.Duration {
	return (*connector.client).PTTL(connector.context, key).Val()
}

// Delete the key, use it only for tests
func (connector *DbConnector) delete(key string) {
	(*connector.client).Del(connector.context, key)
}
