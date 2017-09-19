package redis

import (
	"fmt"
	"net"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
	"gopkg.in/tomb.v2"
)

// DbConnector contains redis pool
type DbConnector struct {
	pool            *redis.Pool
	logger          moira.Logger
	retentionCache  *cache.Cache
	metricsCache    *cache.Cache
	messengersCache *cache.Cache
	sync            *redsync.Redsync
}

// NewDatabase creates Redis pool based on config
func NewDatabase(logger moira.Logger, config Config) *DbConnector {
	pool := newRedisPool(fmt.Sprintf("%s:%s", config.Host, config.Port))
	db := DbConnector{
		pool:            pool,
		logger:          logger,
		retentionCache:  cache.New(time.Minute, time.Minute*60),
		metricsCache:    cache.New(time.Minute, time.Minute*60),
		messengersCache: cache.New(cache.NoExpiration, cache.DefaultExpiration),
		sync:            redsync.New([]redsync.Pool{pool}),
	}
	return &db
}

func newRedisPool(redisURI string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     500,
		MaxActive:   1000,
		IdleTimeout: 5 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisURI, redis.DialConnectTimeout(1*time.Second), redis.DialReadTimeout(1*time.Second), redis.DialWriteTimeout(1*time.Second))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if t.Add(3 * time.Second).Before(time.Now()) {
				_, err := c.Do("PING")
				return err
			}
			return nil
		},
	}
}

func (connector *DbConnector) makePubSubConnection(key string) (*redis.PubSubConn, error) {
	c := connector.pool.Get()
	if c.Err() != nil {
		return nil, c.Err()
	}
	psc := redis.PubSubConn{Conn: c}
	if err := psc.Subscribe(key); err != nil {
		return nil, fmt.Errorf("Failed to subscribe to '%s', error: %v", key, err)
	}
	return &psc, nil
}

func (connector *DbConnector) manageSubscriptions(tomb *tomb.Tomb) (<-chan []byte, error) {
	psc, err := connector.makePubSubConnection(metricEventKey)
	if err != nil {
		return nil, err
	}

	go func() {
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from '%s' redis channel...", metricEventKey)
		psc.Unsubscribe()
	}()

	dataChan := make(chan []byte)
	go func() {
		defer psc.Close()
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				dataChan <- n.Data
			case redis.Subscription:
				if n.Kind == "subscribe" {
					connector.logger.Infof("Subscribe to %s channel, current subscriptions is %v", n.Channel, n.Count)
				} else if n.Kind == "unsubscribe" {
					connector.logger.Infof("Unsubscribe from %s channel, current subscriptions is %v", n.Channel, n.Count)
					if n.Count == 0 {
						connector.logger.Infof("No more subscriptions, exit...")
						close(dataChan)
						return
					}
				}
			case *net.OpError:
				connector.logger.Info("psc.Receive() returned *net.OpError, reconnecting")
				newPsc, err := connector.makePubSubConnection(metricEventKey)
				if err != nil {
					connector.logger.Errorf("Failed to reconnect to subscription: %v", err)
					time.Sleep(time.Second * 5)
					continue
				}
				psc = newPsc
				time.Sleep(time.Second * 5)
			default:
				connector.logger.Errorf("Can not receive message of type '%T': %v", n, n)
				time.Sleep(time.Second * 5)
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
