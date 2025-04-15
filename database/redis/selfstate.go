package redis

import (
	"encoding/json"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// UpdateMetricsHeartbeat increments redis counter.
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := *connector.client
	err := c.Incr(connector.context, selfStateMetricsHeartbeatKey).Err()

	return err
}

// GetMetricsUpdatesCount return metrics count received by Moira-Filter.
func (connector *DbConnector) GetMetricsUpdatesCount() (int64, error) {
	c := *connector.client

	ts, err := c.Get(connector.context, selfStateMetricsHeartbeatKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return ts, err
}

// GetChecksUpdatesCount return checks count by Moira-Checker.
func (connector *DbConnector) GetChecksUpdatesCount() (int64, error) {
	c := *connector.client

	ts, err := c.Get(connector.context, selfStateChecksCounterKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return ts, err
}

// GetRemoteChecksUpdatesCount return remote checks count by Moira-Checker.
func (connector *DbConnector) GetRemoteChecksUpdatesCount() (int64, error) {
	c := *connector.client

	ts, err := c.Get(connector.context, selfStateRemoteChecksCounterKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return ts, err
}

// GetPrometheusChecksUpdatesCount return remote checks count by Moira-Checker.
func (connector *DbConnector) GetPrometheusChecksUpdatesCount() (int64, error) {
	c := *connector.client

	ts, err := c.Get(connector.context, selfStatePrometheusChecksCounterKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}

	return ts, err
}

// GetNotifierState return current notifier state: <OK|ERROR>.
func (connector *DbConnector) GetNotifierState() (moira.NotifierState, error) {
	c := *connector.client
	defaultState := moira.NotifierState{
		State: moira.SelfStateERROR,
		Actor: moira.SelfStateActorManual,
	}

	getResult := c.Get(connector.context, selfStateNotifierHealth)
	if errors.Is(getResult.Err(), redis.Nil) {
		state := moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}

		err := connector.setNotifierState(state)
		if err != nil {
			return defaultState, err
		}

		return state, err
	}

	state, err := reply.NotifierState(getResult)
	if err != nil {
		state := moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}
		err = connector.setNotifierState(state) // NOTE: It's used to migrate from old dto to new
		return state, err
	}

	return state, err
}

// SetNotifierState update current notifier state: <OK|ERROR>.
func (connector *DbConnector) SetNotifierState(actor, state string) error {
	err := connector.setNotifierState(moira.NotifierState{
		State: state,
		Actor: actor,
	})

	return err
}

func (connector *DbConnector) setNotifierState(dto moira.NotifierState) error {
	c := *connector.client

	state, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return c.Set(connector.context, selfStateNotifierHealth, state, redis.KeepTTL).Err()
}

var (
	selfStateMetricsHeartbeatKey        = "moira-selfstate:metrics-heartbeat"
	selfStateChecksCounterKey           = "moira-selfstate:checks-counter"
	selfStateRemoteChecksCounterKey     = "moira-selfstate:remote-checks-counter"
	selfStatePrometheusChecksCounterKey = "moira-selfstate:prometheus-checks-counter"
	selfStateNotifierHealth             = "moira-selfstate:notifier-health"
)
