package redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
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

// GetRemoteChecksUpdatesCount return remote checks count by Moira-Checker
func (connector *DbConnector) GetRemoteChecksUpdatesCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", selfStateRemoteChecksCounterKey))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetNotifierState return current notifier state: <OK|ERROR>
func (connector *DbConnector) GetNotifierState() (string, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.String(c.Do("GET", selfStateNotifierHealth))
	if err == redis.ErrNil {
		ts = moira.SelfStateOK
		err = connector.SetNotifierState(ts)
	} else if err != nil {
		ts = moira.SelfStateERROR
	}
	return ts, err
}

// SetNotifierState update current notifier state: <OK|ERROR>
func (connector *DbConnector) SetNotifierState(health string) error {
	c := connector.pool.Get()
	defer c.Close()

	return c.Send("SET", selfStateNotifierHealth, health)
}

func (connector *DbConnector) GetNotifierMessage() (string, error) {
	c := connector.pool.Get()
	defer c.Close()
	message, err := redis.String(c.Do("GET", selfStateNotifierMessage))
	if err == redis.ErrNil {
		message = dto.DefaultMessage
	}
	return message, err
}

func (connector *DbConnector) SetNotifierMessage(message string) error {
	c := connector.pool.Get()
	defer c.Close()

	return c.Send("SET", selfStateNotifierMessage, message)
}

var selfStateMetricsHeartbeatKey = "moira-selfstate:metrics-heartbeat"
var selfStateChecksCounterKey = "moira-selfstate:checks-counter"
var selfStateRemoteChecksCounterKey = "moira-selfstate:remote-checks-counter"
var selfStateNotifierHealth = "moira-selfstate:notifier-health"
var selfStateNotifierMessage = "moira-selfstate:notifier-message"
