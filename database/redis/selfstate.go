package redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
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
func (connector *DbConnector) GetNotifierState() (string, string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var state, message string
	stateArgs := []interface{} {selfStateNotifierHealth, selfStateNotifierMessage }
	values, err := redis.Strings(c.Do("MGET", stateArgs...))
	if err != nil {
		state = moira.SelfStateERROR
		message = moira.SelfStateErrorMessage
	} else {
		state = values[0]
		message = values[1]
		if state == "" && message == "" {
			state = moira.SelfStateOK
			message = moira.SelfStateOKMessage
			err = connector.SetNotifierState(state, message)
		} else if state == moira.SelfStateERROR && message == "" {
			message = moira.SelfStateErrorMessage
		}
	}
	return state, message, err
}

// SetNotifierState update current notifier state (<OK|ERROR>) and the appropriate message
func (connector *DbConnector) SetNotifierState(state, message string) error {
	if message == "" && state == moira.SelfStateERROR{
		message = moira.SelfStateErrorMessage
	}
	c := connector.pool.Get()
	defer c.Close()

	return c.Send("MSET", selfStateNotifierHealth, state, selfStateNotifierMessage, message)
}

var selfStateMetricsHeartbeatKey = "moira-selfstate:metrics-heartbeat"
var selfStateChecksCounterKey = "moira-selfstate:checks-counter"
var selfStateRemoteChecksCounterKey = "moira-selfstate:remote-checks-counter"
var selfStateNotifierHealth = "moira-selfstate:notifier-health"
var selfStateNotifierMessage = "moira-selfstate:notifier-message"
