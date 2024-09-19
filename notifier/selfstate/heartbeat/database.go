package heartbeat

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
)

var _ Heartbeater = (*databaseHeartbeater)(nil)

type DatabaseHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	RedisDisconnectDelay time.Duration `validate:"required,gt=0"`
}

func (cfg DatabaseHeartbeaterConfig) validate() error {
	validator := validator.New()
	return validator.Struct(cfg)
}

type databaseHeartbeater struct {
	*heartbeaterBase

	cfg DatabaseHeartbeaterConfig
}

func NewDatabaseHeartbeater(cfg DatabaseHeartbeaterConfig, base *heartbeaterBase) (*databaseHeartbeater, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("database heartbeater configuration error: %w", err)
	}

	return &databaseHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

func (heartbeater *databaseHeartbeater) Check() (State, error) {
	now := heartbeater.clock.NowUTC()

	_, err := heartbeater.database.GetChecksUpdatesCount()
	if err == nil {
		heartbeater.lastSuccessfulCheck = now
		return StateOK, nil
	}

	if now.Sub(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.RedisDisconnectDelay {
		return StateError, nil
	}

	return StateOK, err
}

func (heartbeater databaseHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

func (heartbeater databaseHeartbeater) NeedToCheckOthers() bool {
	return heartbeater.cfg.NeedToCheckOthers
}

func (databaseHeartbeater) Type() moira.EmergencyContactType {
	return moira.EmergencyTypeRedisDisconnected
}

func (heartbeater databaseHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
