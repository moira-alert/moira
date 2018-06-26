package redis

import (
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira/notifier/selfstate"
)

// UpdateMetricsHeartbeat increments redis counter
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := connector.pool.Get()
	defer c.Close()
	err := c.Send("INCR", selfStateMetricsHeartbeatKey)
	return err
}

// GetMetricsUpdatesCount return metrics count received by Moira-Filter
func (connector *DbConnector) GetMetricsUpdatesCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", selfStateMetricsHeartbeatKey))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetChecksUpdatesCount return checks count by Moira-Checker
func (connector *DbConnector) GetChecksUpdatesCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", selfStateChecksCounterKey))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

func (connector *DbConnector) GetNotifierState() (string, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.String(c.Do("GET", selfStateNotifierHealth))
	if err == redis.ErrNil {
		ts = selfstate.OK
		err = connector.SetNotifierState(ts)
	}
	return ts, err
}

func (connector *DbConnector) SetNotifierState(health string) error {
	c := connector.pool.Get()
	defer c.Close()

	return c.Send("SET", selfStateNotifierHealth, health)
}

var selfStateMetricsHeartbeatKey = "moira-selfstate:metrics-heartbeat"
var selfStateChecksCounterKey = "moira-selfstate:checks-counter"
var selfStateNotifierHealth = "moira-selfstate:notifier-health"
