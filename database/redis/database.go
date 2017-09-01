package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/patrickmn/go-cache"
	"time"
)

// DbConnector contains redis pool
type DbConnector struct {
	pool           *redis.Pool
	logger         moira.Logger
	metrics        *graphite.DatabaseMetrics
	retentionCache *cache.Cache
}

// NewDatabase creates Redis pool based on config
func NewDatabase(logger moira.Logger, config Config, metrics *graphite.DatabaseMetrics) *DbConnector {
	db := DbConnector{
		pool:           newRedisPool(fmt.Sprintf("%s:%s", config.Host, config.Port), config.DBID),
		logger:         logger,
		metrics:        metrics,
		retentionCache: cache.New(time.Minute, time.Minute*30),
	}
	return &db
}

func newRedisPool(redisURI string, dbID ...int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisURI)
			if err != nil {
				return nil, err
			}
			if len(dbID) > 0 {
				c.Do("SELECT", dbID[0])
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
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
			}
		}
	}()
	return dataChan
}
