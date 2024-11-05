package heartbeat

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

// Verify that databaseHeartbeater matches the Heartbeater interface.
var _ Heartbeater = (*databaseHeartbeater)(nil)

// DatabaseHeartbeaterConfig structure describing the databaseHeartbeater configuration.
type DatabaseHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	RedisDisconnectDelay time.Duration `validate:"required,gt=0"`
}

type databaseHeartbeater struct {
	*heartbeaterBase

	cfg DatabaseHeartbeaterConfig
}

// NewDatabaseHeartbeater is a function that creates a new databaseHeartbeater.
func NewDatabaseHeartbeater(cfg DatabaseHeartbeaterConfig, base *heartbeaterBase) (*databaseHeartbeater, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
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

// Type is a function that returns the current heartbeat type.
func (databaseHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatDatabase
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater databaseHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
