package redis

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/FZambia/go-sentinel"
	"github.com/garyburd/redigo/redis"
	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
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
	servicesCache        *cache.Cache
	sync                 *redsync.Redsync
	source               DBSource
}

// NewDatabase creates Redis pool based on config
func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	pool := newRedisPool(logger, config)
	return &DbConnector{
		pool:                 pool,
		logger:               logger,
		retentionCache:       cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		retentionSavingCache: cache.New(cache.NoExpiration, cache.DefaultExpiration),
		metricsCache:         cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		servicesCache:        cache.New(cacheValueExpirationDuration, cacheCleanupInterval),
		sync:                 redsync.New([]redsync.Pool{pool}),
		source:               source,
	}
}

func newRedisPool(logger moira.Logger, config Config) *redis.Pool {
	serverAddr := net.JoinHostPort(config.Host, config.Port)
	useSentinel := config.MasterName != "" && len(config.SentinelAddresses) > 0
	if !useSentinel {
		logger.Infof("Redis: %v, DbID: %v", serverAddr, config.DBID)
	} else {
		logger.Infof("Redis: Sentinel for name: %v, DbID: %v", config.MasterName, config.DBID)
	}
	sntnl, err := createSentinel(logger, config, useSentinel)
	if err != nil {
		logger.Error(err)
		return nil
	}

	var lastMu sync.Mutex
	var lastMaster string

	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			if sntnl != nil {
				serverAddr, err = sntnl.MasterAddr()
				if err != nil {
					return nil, err
				}
				lastMu.Lock()
				if serverAddr != lastMaster {
					logger.Infof("Redis master discovered: %s", serverAddr)
					lastMaster = serverAddr
				}
				lastMu.Unlock()
			}
			return redis.Dial("tcp", serverAddr, redis.DialDatabase(config.DBID))
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if useSentinel {
				if !sentinel.TestRole(c, "master") {
					return fmt.Errorf("failed master role check")
				}
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func createSentinel(logger moira.Logger, config Config, useSentinel bool) (*sentinel.Sentinel, error) {
	if useSentinel {
		sntnl := &sentinel.Sentinel{
			Addrs:      config.SentinelAddresses,
			MasterName: config.MasterName,
			Dial: func(addr string) (redis.Conn, error) {
				timeout := 300 * time.Millisecond
				return redis.Dial(
					"tcp",
					addr,
					redis.DialConnectTimeout(timeout),
					redis.DialReadTimeout(timeout),
					redis.DialWriteTimeout(timeout),
				)
			},
		}

		// Periodically discover new Sentinels.
		go func() {
			if err := sntnl.Discover(); err != nil {
				logger.Error(err)
			}
			checkTicker := time.NewTicker(30 * time.Second)
			for {
				<-checkTicker.C
				if err := sntnl.Discover(); err != nil {
					logger.Error(err)
				}
			}
		}()
		return sntnl, nil
	}
	return nil, nil
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
