package redis

import (
	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

// UpdateMetricsHeartbeat increments redis counter
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := *connector.client
	err := c.Incr(connector.context, selfStateMetricsHeartbeatKey).Err()
	return err
}

// GetMetricsUpdatesCount return metrics count received by Moira-Filter
func (connector *DbConnector) GetMetricsUpdatesCount() (int64, error) {
	c := *connector.client
	ts, err := c.Get(connector.context, selfStateMetricsHeartbeatKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return ts, err
}

// GetChecksUpdatesCount return checks count by Moira-Checker
func (connector *DbConnector) GetChecksUpdatesCount() (int64, error) {
	c := *connector.client
	ts, err := c.Get(connector.context, selfStateChecksCounterKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return ts, err
}

// GetRemoteChecksUpdatesCount return remote checks count by Moira-Checker
func (connector *DbConnector) GetRemoteChecksUpdatesCount() (int64, error) {
	c := *connector.client
	ts, err := c.Get(connector.context, selfStateRemoteChecksCounterKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return ts, err
}

// GetNotifierState return current notifier state: <OK|ERROR>
func (connector *DbConnector) GetNotifierState() (string, error) {
	c := *connector.client
	ts, err := c.Get(connector.context, selfStateNotifierHealth).Result()
	if err == redis.Nil {
		ts = moira.SelfStateOK
		err = connector.SetNotifierState(ts)
	} else if err != nil {
		ts = moira.SelfStateERROR
	}
	return ts, err
}

// SetNotifierState update current notifier state: <OK|ERROR>
func (connector *DbConnector) SetNotifierState(health string) error {
	c := *connector.client
	return c.Set(connector.context, selfStateNotifierHealth, health, redis.KeepTTL).Err()
}

var selfStateMetricsHeartbeatKey = "moira-selfstate:metrics-heartbeat"
var selfStateChecksCounterKey = "moira-selfstate:checks-counter"
var selfStateRemoteChecksCounterKey = "moira-selfstate:remote-checks-counter"
var selfStateNotifierHealth = "moira-selfstate:notifier-health"
