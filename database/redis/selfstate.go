package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
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

	getResult := c.Get(connector.context, selfStateNotifierHealth)
	if errors.Is(getResult.Err(), redis.Nil) {
		state := moira.NotifierState{
			State:     moira.SelfStateOK,
			Actor:     moira.SelfStateActorManual,
			Timestamp: connector.Clock.NowUnix(),
		}

		err := connector.setNotifierState(state)
		if err != nil {
			return errorState(connector.Clock), err
		}

		return state, err
	}

	state, err := reply.NotifierState(getResult)
	if err != nil {
		state := moira.NotifierState{
			State:     moira.SelfStateOK,
			Actor:     moira.SelfStateActorManual,
			Timestamp: connector.Clock.NowUnix(),
		}

		err = connector.setNotifierState(state) // NOTE: It's used to migrate from old dto to new
		if err != nil {
			return moira.NotifierState{
				State:     moira.SelfStateERROR,
				Actor:     moira.SelfStateActorAutomatic,
				Timestamp: connector.Clock.NowUnix(),
			}, err
		}

		return state, err
	}

	return state, err
}

// SetNotifierState update current notifier state: <OK|ERROR>.
func (connector *DbConnector) SetNotifierState(actor, state string) error {
	err := connector.setNotifierState(moira.NotifierState{
		State:     state,
		Actor:     actor,
		Timestamp: connector.Clock.NowUnix(),
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

// GetNotifierStateForSource returns state for a given metric source cluster.
func (connector *DbConnector) GetNotifierStateForSource(clusterKey moira.ClusterKey) (moira.NotifierState, error) {
	if !slices.Contains(connector.clusterList, clusterKey) {
		return errorState(connector.Clock), fmt.Errorf("unknown cluster '%s'", clusterKey.String())
	}

	c := *connector.client

	stateCmd := c.Get(connector.context, makeSelfStateNotifierStateForSource(clusterKey))

	state, err := reply.NotifierState(stateCmd)
	if err != nil && !errors.Is(err, database.ErrNil) {
		return errorState(connector.Clock), err
	}

	if errors.Is(err, database.ErrNil) {
		// If state for cluster was never set, set OK by default
		return okState(connector.Clock), nil
	}

	return state, nil
}

// GetNotifierStateForSources returns state for all metric source clusters.
func (connector *DbConnector) GetNotifierStateForSources() (map[moira.ClusterKey]moira.NotifierState, error) {
	c := *connector.client

	statesCmd := make([]*redis.StringCmd, 0, len(connector.clusterList))

	_, _ = c.TxPipelined(connector.context, func(p redis.Pipeliner) error {
		for _, cluster := range connector.clusterList {
			statesCmd = append(statesCmd, p.Get(connector.context, makeSelfStateNotifierStateForSource(cluster)))
		}

		return nil
	})

	result := make(map[moira.ClusterKey]moira.NotifierState, len(connector.clusterList))

	for i, cluster := range connector.clusterList {
		state, err := reply.NotifierState(statesCmd[i])
		if err != nil && !errors.Is(err, database.ErrNil) {
			return nil, err
		}

		if errors.Is(err, database.ErrNil) {
			// If state for cluster was never set, set OK by default
			result[cluster] = okState(connector.Clock)
		} else {
			result[cluster] = state
		}
	}

	return result, nil
}

// SetNotifierStateForSource saves state for given metric source cluster.
func (connector *DbConnector) SetNotifierStateForSource(clusterKey moira.ClusterKey, actor, state string) error {
	if !slices.Contains(connector.clusterList, clusterKey) {
		return fmt.Errorf("unknown cluster '%s'", clusterKey.String())
	}

	c := *connector.client

	currentState := moira.NotifierState{
		State:     state,
		Actor:     actor,
		Timestamp: connector.Clock.NowUnix(),
	}

	bytes, err := json.Marshal(currentState)
	if err != nil {
		return err
	}

	saveCmd := c.Set(connector.context, makeSelfStateNotifierStateForSource(clusterKey), bytes, redis.KeepTTL)

	err = saveCmd.Err()
	if err != nil {
		return err
	}

	return nil
}

func errorState(clock moira.Clock) moira.NotifierState {
	return moira.NotifierState{
		State:     moira.SelfStateERROR,
		Actor:     moira.SelfStateActorManual,
		Timestamp: clock.NowUnix(),
	}
}

func okState(clock moira.Clock) moira.NotifierState {
	return moira.NotifierState{
		State:     moira.SelfStateOK,
		Actor:     moira.SelfStateActorManual,
		Timestamp: clock.NowUnix(),
	}
}

func makeSelfStateNotifierStateForSource(clusterKey moira.ClusterKey) string {
	return selfStateNotifierStateForSource + ":" + clusterKey.String()
}

var (
	selfStateMetricsHeartbeatKey        = "moira-selfstate:metrics-heartbeat"
	selfStateChecksCounterKey           = "moira-selfstate:checks-counter"
	selfStateRemoteChecksCounterKey     = "moira-selfstate:remote-checks-counter"
	selfStatePrometheusChecksCounterKey = "moira-selfstate:prometheus-checks-counter"
	selfStateNotifierHealth             = "moira-selfstate:notifier-health"
	selfStateNotifierStateForSource     = "moira-selfstate:notifier-state-for-source"
)
