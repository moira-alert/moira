package heartbeat

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

// Verify that notifierHeartbeater matches the Heartbeater interface.
var _ Heartbeater = (*notifierHeartbeater)(nil)

// NotifierHeartbeaterConfig structure describing the notifierHeartbeater configuration.
type NotifierHeartbeaterConfig struct {
	HeartbeaterBaseConfig
}

type notifierHeartbeater struct {
	*heartbeaterBase

	cfg NotifierHeartbeaterConfig
}

// NewNotifierHeartbeater is a function that creates a new notifierHeartbeater.
func NewNotifierHeartbeater(cfg NotifierHeartbeaterConfig, base *heartbeaterBase) (*notifierHeartbeater, error) {
	return &notifierHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

// Check is a function that returns the state of the notifier.
func (heartbeater *notifierHeartbeater) Check() (State, error) {
	notifierState, err := heartbeater.database.GetNotifierState()
	if err != nil {
		return StateError, err
	}

	if notifierState != moira.SelfStateOK {
		return StateError, nil
	}

	return StateOK, nil
}

// NeedTurnOffNotifier is a function that checks to see if the notifier needs to be turned off.
func (heartbeater *notifierHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

// Type is a function that returns the current heartbeat type.
func (notifierHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatNotifier
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater notifierHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
