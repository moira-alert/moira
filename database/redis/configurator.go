package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"time"
)

// Init creates Redis pool based on config
func Init(logger *moira_alert.Logger, config Config) *DbConnector {
	db := DbConnector{
		Pool:   newRedisPool(fmt.Sprintf("%s:%s", config.Host, config.Port), config.DBID),
		logger: *logger,
	}
	return &db
}

// NewRedisPool creates Redis pool
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
