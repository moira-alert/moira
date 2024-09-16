package heartbeat

import (
	"github.com/moira-alert/moira"
)

var _ Heartbeater = (*notifierHeartbeater)(nil)

type NotifierHeartbeaterConfig struct {
	HeartbeaterBaseConfig
}

type notifierHeartbeater struct {
	*heartbeaterBase

	cfg NotifierHeartbeaterConfig
}

func NewNotifierHeartbeater(cfg NotifierHeartbeaterConfig, base *heartbeaterBase) (*notifierHeartbeater, error) {
	return &notifierHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

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

func (heartbeater *notifierHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

func (heartbeater *notifierHeartbeater) NeedToCheckOthers() bool {
	return heartbeater.cfg.NeedToCheckOthers
}

func (notifierHeartbeater) Type() moira.EmergencyContactType {
	return moira.EmergencyTypeNotifierOff
}

func (heartbeater notifierHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
