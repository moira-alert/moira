package heartbeat

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

var (
	localClusterKey = moira.DefaultLocalCluster

	// Verify that filterHeartbeater matches the Heartbeater interface.
	_ Heartbeater = (*filterHeartbeater)(nil)
)

// FilterHeartbeaterConfig structure describing the filterHeartbeater configuration.
type FilterHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	MetricReceivedDelay time.Duration `validate:"required,gt=0"`
}

type filterHeartbeater struct {
	*heartbeaterBase

	cfg              FilterHeartbeaterConfig
	lastMetricsCount int64
}

// NewFilterHeartbeater is a function that creates a new filterHeartbeater.
func NewFilterHeartbeater(cfg FilterHeartbeaterConfig, base *heartbeaterBase) (*filterHeartbeater, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("filter heartheater configuration error: %w", err)
	}

	return &filterHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

// Check is a function that checks that filters accept metrics and that their number of metrics is not constant.
func (heartbeater *filterHeartbeater) Check() (State, error) {
	triggersCount, err := heartbeater.database.GetTriggersToCheckCount(localClusterKey)
	if err != nil {
		return StateError, err
	}

	metricsCount, err := heartbeater.database.GetMetricsUpdatesCount()
	if err != nil {
		return StateError, err
	}

	now := heartbeater.clock.NowUTC()
	if heartbeater.lastMetricsCount != metricsCount || triggersCount == 0 {
		heartbeater.lastMetricsCount = metricsCount
		heartbeater.lastSuccessfulCheck = now
		return StateOK, nil
	}

	if now.Sub(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.MetricReceivedDelay {
		return StateError, nil
	}

	return StateOK, nil
}

// NeedTurnOffNotifier is a function that checks to see if the notifier needs to be turned off.
func (heartbeater filterHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

// Type is a function that returns the current heartbeat type.
func (filterHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatTypeNotSet
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater filterHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
