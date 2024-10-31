package heartbeat

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira/datatypes"
)

// Verify that databaseHeartbeater matches the Heartbeater interface.
var _ Heartbeater = (*databaseHeartbeater)(nil)

// DatabaseHeartbeaterConfig structure describing the databaseHeartbeater configuration.
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

// NewDatabaseHeartbeater is a function that creates a new databaseHeartbeater.
func NewDatabaseHeartbeater(cfg DatabaseHeartbeaterConfig, base *heartbeaterBase) (*databaseHeartbeater, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("database heartbeater configuration error: %w", err)
	}

	return &databaseHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

// Check is a function that checks if the database is working correctly.
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

// NeedTurnOffNotifier is a function that checks to see if the notifier needs to be turned off.
func (heartbeater databaseHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

// Type is a function that returns the current heartbeat type.
func (databaseHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HearbeatTypeNotSet
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater databaseHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
