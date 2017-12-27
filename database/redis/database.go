package redis

import (
	"fmt"
	"net"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
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
	pool := newRedisPool(fmt.Sprintf("%s:%s", config.Host, config.Port), config.DBID)
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

func (connector *DbConnector) makePubSubConnection(channel string) (*redis.PubSubConn, error) {
	c := connector.pool.Get()
	psc := redis.PubSubConn{Conn: c}
	if err := psc.Subscribe(channel); err != nil {
		return nil, fmt.Errorf("Failed to subscribe to '%s', error: %v", channel, err)
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

	dataChan := make(chan []byte)
	go func() {
		defer psc.Close()
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
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
				connector.logger.Info("psc.Receive() returned *net.OpError: %s. Reconnecting...", n.Err.Error())
				newPsc, err := connector.makePubSubConnection(metricEventKey)
				if err != nil {
					connector.logger.Errorf("Failed to reconnect to subscription: %v", err)
					<-time.After(5 * time.Second)
					continue
				}
				psc = newPsc
				<-time.After(5 * time.Second)
			default:
				connector.logger.Errorf("Can not receive message of type '%T': %v", n, n)
				<-time.After(5 * time.Second)
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
