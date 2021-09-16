package goredis

import (
	"context"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"
)

const pubSubWorkerChannelSize = 16384

const (
	cacheCleanupInterval         = time.Minute * 60
	cacheValueExpirationDuration = time.Minute
)

const (
	receiveErrorSleepDuration = time.Second
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
}

func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	client := newClient(&redis.UniversalOptions{
		MasterName:     config.MasterName,
		Addrs:          config.Addrs,
		RouteByLatency: true, // for Sentinel or Redis Cluster only to route readonly commands to slave nodes
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
	}
	return &connector
}

// newClient returns a new multi client. The type of the returned client depends
// on the following conditions:
//
// 1. If the MasterName option is specified, a sentinel-backed FailoverClusterClient is returned.
// 2. Otherwise, a single-node Client is returned.
// This is modification of go-redis/redis's NewUniversalClient function
func newClient(opts *redis.UniversalOptions) redis.UniversalClient {
	if opts.MasterName != "" {
		return redis.NewFailoverClusterClient(opts.Failover())
	}
	return redis.NewClient(opts.Simple())
}

func (connector *DbConnector) manageSubscriptions(tomb *tomb.Tomb, channel string) (<-chan []byte, error) {
	c := (*connector.client).Subscribe(connector.context, channel)

	go func() {
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from '%s' redis channels...", channel)
		c.Unsubscribe(connector.context) //nolint
	}()

	dataChan := make(chan []byte, pubSubWorkerChannelSize)
	go func() {
		for {
			raw, _ := c.Receive(connector.context)
			switch data := raw.(type) {
			case *redis.Message:
				if len(data.Payload) == 0 {
					continue
				}
				dataChan <- []byte(data.Payload)
			case *redis.Subscription:
				switch data.Kind {
				case "subscribe":
					connector.logger.Infof("Subscribe to %s channel, current subscriptions is %v", data.Channel, data.Count)
				case "unsubscribe":
					connector.logger.Infof("Unsubscribe from %s channel, current subscriptions is %v", data.Channel, data.Count)
					if data.Count == 0 {
						connector.logger.Infof("No more subscriptions, exit...")
						close(dataChan)
						return
					}
				}
			case *net.OpError:
				connector.logger.Infof("psc.Receive() returned *net.OpError: %s. Reconnecting...", data.Err.Error())
				c = (*connector.client).Subscribe(connector.context, channel)
				<-time.After(receiveErrorSleepDuration)
			default:
				connector.logger.Errorf("Can not receive message of type '%T': %v", raw, raw)
				<-time.After(receiveErrorSleepDuration)
			}
		}
	}()

	return dataChan, c.Ping(connector.context)
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
