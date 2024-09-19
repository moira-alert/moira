package heartbeat

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
)

var (
	remoteClusterKey = moira.DefaultGraphiteRemoteCluster

	_ Heartbeater = (*remoteCheckerHeartbeater)(nil)
)

type RemoteCheckerHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	RemoteCheckDelay time.Duration `validate:"required,gt=0"`
}

func (cfg RemoteCheckerHeartbeaterConfig) validate() error {
	validator := validator.New()
	return validator.Struct(cfg)
}

type remoteCheckerHeartbeater struct {
	*heartbeaterBase

	cfg                   RemoteCheckerHeartbeaterConfig
	lastRemoteChecksCount int64
}

func NewRemoteCheckerHeartbeater(cfg RemoteCheckerHeartbeaterConfig, base *heartbeaterBase) (*remoteCheckerHeartbeater, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("remote checker heartbeater configuration error: %w", err)
	}

	return &remoteCheckerHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

func (heartbeater remoteCheckerHeartbeater) Check() (State, error) {
	triggersCount, err := heartbeater.database.GetTriggersToCheckCount(remoteClusterKey)
	if err != nil {
		return StateError, err
	}

	remoteChecksCount, err := heartbeater.database.GetRemoteChecksUpdatesCount()
	if err != nil {
		return StateError, err
	}

	now := heartbeater.clock.NowUTC()
	if heartbeater.lastRemoteChecksCount != remoteChecksCount || triggersCount == 0 {
		heartbeater.lastRemoteChecksCount = remoteChecksCount
		heartbeater.lastSuccessfulCheck = now
		return StateOK, nil
	}

	if now.Sub(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.RemoteCheckDelay {
		return StateError, nil
	}

	return StateOK, nil
}

func (heartbeater remoteCheckerHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

func (heartbeater remoteCheckerHeartbeater) NeedToCheckOthers() bool {
	return heartbeater.cfg.NeedToCheckOthers
}

func (remoteCheckerHeartbeater) Type() moira.EmergencyContactType {
	return moira.EmergencyTypeRemoteCheckerNoTriggerCheck
}

func (heartbeater remoteCheckerHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
