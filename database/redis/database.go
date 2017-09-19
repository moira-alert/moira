package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
	"time"
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
		MaxActive:   500,
		IdleTimeout: 5 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialTimeout("tcp", redisURI, 100*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if t.Add(3 * time.Second).Before(time.Now()) {
				_, err := c.Do("PING")
			}
			return err
		},
	}
}

func (connector *DbConnector) manageSubscriptions(psc redis.PubSubConn) <-chan []byte {
	dataChan := make(chan []byte)
	go func() {
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
			default:
				connector.logger.Errorf("Can not receive message of type '%T': %v", n, n)
				time.Sleep(time.Second * 5)
			}
		}
	}()
	return dataChan
}

// CLEAN DATABASE! USE IT ONLY FOR TESTING!!!
func (connector *DbConnector) flush() {
	c := connector.pool.Get()
	defer c.Close()
	c.Do("FLUSHDB")
}
