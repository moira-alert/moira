package redis

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/moira-alert/moira/clock"

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
	clock                moira.Clock
}

func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		MasterName:   config.MasterName,
		Addrs:        config.Addrs,
		Username:     config.Username,
		Password:     config.Password,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		MaxRetries:   config.MaxRetries,
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
	}
	return &connector
}

// NewTestDatabase use it only for tests
func NewTestDatabase(logger moira.Logger) *DbConnector {
	return NewDatabase(logger, Config{
		Addrs: []string{"0.0.0.0:6379"},
	}, testSource)
}

// NewTestDatabaseWithIncorrectConfig use it only for tests
func NewTestDatabaseWithIncorrectConfig(logger moira.Logger) *DbConnector {
	return NewDatabase(logger, Config{Addrs: []string{"0.0.0.0:0000"}}, testSource)
}

func (connector *DbConnector) makePubSubConnection(channels []string) (*redis.PubSub, error) {
	psc := (*connector.client).Subscribe(connector.context, channels...)
	err := psc.Ping(connector.context)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to '%s', error: %v", channels, err)
	}
	return psc, nil
}

func (connector *DbConnector) manageSubscriptions(tomb *tomb.Tomb, channels []string) (<-chan []byte, error) {
	psc, err := connector.makePubSubConnection(channels)
	if err != nil {
		return nil, err
	}

	go func() {
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from '%s' redis channels...", channels)
		psc.Unsubscribe(connector.context) //nolint
	}()

	dataChan := make(chan []byte, pubSubWorkerChannelSize)
	go func() {
		for {
			raw, err := psc.Receive(connector.context)
			if err != nil {
				if err == io.ErrUnexpectedEOF {
					connector.logger.Infof("Receive() returned error: %s", err.Error())
					continue
				}
				connector.logger.Infof("Receive() returned error: %s. Reconnecting...", err.Error())
				newPsc, err := connector.makePubSubConnection(channels)
				if err != nil {
					connector.logger.ErrorWithError("Failed to reconnect to subscription", err)
					<-time.After(receiveErrorSleepDuration)
					continue
				}
				connector.logger.Info("Reconnected to subscription")
				psc = newPsc
				<-time.After(receiveErrorSleepDuration)
				continue
			}
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
			case *redis.Pong:
				connector.logger.Infof("Received PONG message")
			default:
				connector.logger.Errorb().
					String("message_type", fmt.Sprintf("%T", raw)).
					String("message_content", fmt.Sprintf("%v", raw)).
					Msg("Can not receive message of this type")
				<-time.After(receiveErrorSleepDuration)
			}
		}
	}()

	return dataChan, nil
}

// Flush deletes all the keys of the DB, use it only for tests
func (connector *DbConnector) Flush() {
	client := *connector.client

	switch c := client.(type) {
	case *redis.ClusterClient:
		err := c.ForEachMaster(connector.context, func(ctx context.Context, shard *redis.Client) error {
			return shard.FlushDB(ctx).Err()
		})
		if err != nil {
			return
		}
	default:
		(*connector.client).FlushDB(connector.context)
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
