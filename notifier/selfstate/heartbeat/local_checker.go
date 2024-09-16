package heartbeat

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
)

var _ Heartbeater = (*localCheckerHeartbeater)(nil)

type LocalCheckerHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	LocalCheckDelay time.Duration `validate:"required,gt=0"`
}

func (cfg LocalCheckerHeartbeaterConfig) validate() error {
	validator := validator.New()
	return validator.Struct(cfg)
}

type localCheckerHeartbeater struct {
	*heartbeaterBase

	cfg             LocalCheckerHeartbeaterConfig
	lastChecksCount int64
}

func NewLocalCheckerHeartbeater(cfg LocalCheckerHeartbeaterConfig, base *heartbeaterBase) (*localCheckerHeartbeater, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("local checker heartbeater configuration error: %w", err)
	}

	return &localCheckerHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

func (heartbeater *localCheckerHeartbeater) Check() (State, error) {
	triggersCount, err := heartbeater.database.GetTriggersToCheckCount(localClusterKey)
	if err != nil {
		return StateError, err
	}

	checksCount, err := heartbeater.database.GetChecksUpdatesCount()
	if err != nil {
		return StateError, err
	}

	if heartbeater.lastChecksCount != checksCount || triggersCount == 0 {
		heartbeater.lastChecksCount = checksCount
		heartbeater.lastSuccessfulCheck = heartbeater.clock.NowUTC()
		return StateOK, nil
	}

	if time.Since(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.LocalCheckDelay {
		return StateError, nil
	}

	return StateOK, nil
}

func (heartbeater localCheckerHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

func (heartbeater localCheckerHeartbeater) NeedToCheckOthers() bool {
	return heartbeater.cfg.NeedToCheckOthers
}

func (localCheckerHeartbeater) Type() moira.EmergencyContactType {
	return moira.EmergencyTypeCheckerNoTriggerCheck
}

func (heartbeater localCheckerHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
