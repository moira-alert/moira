package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"time"
)

// NewDatabase creates Redis pool based on config
func NewDatabase(logger moira.Logger, config Config, metrics *graphite.DatabaseMetrics) *DbConnector {
	db := DbConnector{
		pool:    newRedisPool(fmt.Sprintf("%s:%s", config.Host, config.Port), config.DBID),
		logger:  logger,
		metrics: metrics,
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
