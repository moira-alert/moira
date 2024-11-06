package heartbeat

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
	"github.com/moira-alert/moira/metrics"
)

// Verify that notifierHeartbeater matches the Heartbeater interface.
var _ Heartbeater = (*notifierHeartbeater)(nil)

// NotifierHeartbeaterConfig structure describing the notifierHeartbeater configuration.
type NotifierHeartbeaterConfig struct {
	HeartbeaterBaseConfig
}

type notifierHeartbeater struct {
	*heartbeaterBase

	metrics *metrics.HeartBeatMetrics
	cfg     NotifierHeartbeaterConfig
}

// NewNotifierHeartbeater is a function that creates a new notifierHeartbeater.
func NewNotifierHeartbeater(
	cfg NotifierHeartbeaterConfig,
	base *heartbeaterBase,
	metrics *metrics.HeartBeatMetrics,
) (*notifierHeartbeater, error) {
	return &notifierHeartbeater{
		cfg:             cfg,
		heartbeaterBase: base,
		metrics:         metrics,
	}, nil
}

// Check is a function that returns the state of the notifier.
func (heartbeater *notifierHeartbeater) Check() (State, error) {
	notifierState, err := heartbeater.database.GetNotifierState()
	if err != nil {
		heartbeater.metrics.MarkNotifierIsAlive(false)
		return StateError, err
	}

	if notifierState != moira.SelfStateOK {
		heartbeater.metrics.MarkNotifierIsAlive(false)
		return StateError, nil
	}

	heartbeater.metrics.MarkNotifierIsAlive(true)

	return StateOK, nil
}

// Type is a function that returns the current heartbeat type.
func (notifierHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatNotifier
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater notifierHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
