package redis

import (
	"fmt"
	"net"
	"time"

	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
)

const pubSubWorkerChannelSize = 16384
const dialTimeout = time.Millisecond * 500

const (
	cacheCleanupInterval         = time.Minute * 60
	cacheValueExpirationDuration = time.Minute
)

const (
	receiveErrorSleepDuration = time.Second
)

// DBSource is type for describing who create database instance
type DBSource string

// All types of database instances users
const (
	API      DBSource = "API"
	Checker  DBSource = "Checker"
	Filter   DBSource = "Filter"
	Notifier DBSource = "Notifier"
	Cli      DBSource = "Cli"
)

// DbConnector contains redis pool
type DbConnector struct {
	pool                 *redis.Pool
	logger               moira.Logger
	retentionCache       *cache.Cache
	retentionSavingCache *cache.Cache
	metricsCache         *cache.Cache
	sync                 *redsync.Redsync
	source               DBSource
}

// NewDatabase creates Redis pool based on config
func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	poolDialer := newPoolDialer(logger, config)

	pool := &redis.Pool{
		MaxIdle:      config.ConnectionLimit,
		MaxActive:    config.ConnectionLimit,
		Wait:         true,
		IdleTimeout:  240 * time.Second,
		Dial:         poolDialer.Dial,
		TestOnBorrow: poolDialer.Test,
	}
	syncPool := &redis.Pool{
		MaxIdle:      3,
		MaxActive:    10,
		Wait:         true,
		IdleTimeout:  240 * time.Second,
		Dial:         poolDialer.Dial,
		TestOnBorrow: poolDialer.Test,
	}

	return &DbConnector{
		pool:                 pool,
		logger:               logger,
		retentionCache:       cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		retentionSavingCache: cache.New(cache.NoExpiration, cache.DefaultExpiration),
		metricsCache:         cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		sync:                 redsync.New([]redsync.Pool{syncPool}),
		source:               source,
	}
}

// NewTestDatabase use it only for tests
func NewTestDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	config.DB = 1
	return NewDatabase(logger,config,source)
}

func newPoolDialer(logger moira.Logger, config Config) PoolDialer {
	if config.MasterName != "" && len(config.SentinelAddresses) > 0 {
		logger.Infof("Redis: Sentinel for name: %v, DB: %v", config.MasterName, config.DB)
		return NewSentinelPoolDialer(
			logger,
			SentinelPoolDialerConfig{
				MasterName:        config.MasterName,
				SentinelAddresses: config.SentinelAddresses,
				DB:                config.DB,
				DialTimeout:       dialTimeout,
			},
		)
	}

	serverAddr := net.JoinHostPort(config.Host, config.Port)
	logger.Infof("Redis: %v, DB: %v", serverAddr, config.DB)
	return &DirectPoolDialer{
		serverAddress: serverAddr,
		db:            config.DB,
		dialTimeout:   dialTimeout,
	}
}

func (connector *DbConnector) makePubSubConnection(channel string) (*redis.PubSubConn, error) {
	c := connector.pool.Get()
	psc := redis.PubSubConn{Conn: c}
	if err := psc.Subscribe(channel); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to subscribe to '%s', error: %v", channel, err)
	}
	return &psc, nil
}

func (connector *DbConnector) manageSubscriptions(tomb *tomb.Tomb, channel string) (<-chan []byte, error) {
	psc, err := connector.makePubSubConnection(channel)
	if err != nil {
		return nil, err
	}

	go func() {
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from '%s' redis channels...", channel)
		psc.Unsubscribe()
	}()

	dataChan := make(chan []byte, pubSubWorkerChannelSize)
	go func() {
		defer psc.Close()
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				if len(n.Data) == 0 {
					continue
				}
				dataChan <- n.Data
			case redis.Subscription:
				switch n.Kind {
				case "subscribe":
					connector.logger.Infof("Subscribe to %s channel, current subscriptions is %v", n.Channel, n.Count)
				case "unsubscribe":
					connector.logger.Infof("Unsubscribe from %s channel, current subscriptions is %v", n.Channel, n.Count)
					if n.Count == 0 {
						connector.logger.Infof("No more subscriptions, exit...")
						close(dataChan)
						return
					}
				}
			case *net.OpError:
				connector.logger.Infof("psc.Receive() returned *net.OpError: %s. Reconnecting...", n.Err.Error())
				newPsc, err := connector.makePubSubConnection(metricEventKey)
				if err != nil {
					connector.logger.Errorf("Failed to reconnect to subscription: %v", err)
					<-time.After(receiveErrorSleepDuration)
					continue
				}
				psc = newPsc
				<-time.After(receiveErrorSleepDuration)
			default:
				connector.logger.Errorf("Can not receive message of type '%T': %v", n, n)
				<-time.After(receiveErrorSleepDuration)
			}
		}
	}()
	return dataChan, nil
}

// CLEAN DATABASE! USE IT ONLY FOR TESTING!!!
func (connector *DbConnector) flush() {
	c := connector.pool.Get()
	defer c.Close()
	c.Do("FLUSHDB")
}

// GET KEY TTL! USE IT ONLY FOR TESTING!!!
func (connector *DbConnector) getTTL(key string) int {
	c := connector.pool.Get()
	defer c.Close()
	ttl, err := redis.Int(c.Do("PTTL", key))
	if err != nil {
		return 0
	}
	return ttl
}

// DELETE KEY! USE IT ONLY FOR TESTING!!!
func (connector *DbConnector) delete(key string) {
	c := connector.pool.Get()
	defer c.Close()
	c.Do("DEL", key)
}
