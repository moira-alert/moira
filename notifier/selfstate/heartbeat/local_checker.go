package heartbeat

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

// Verify that localCheckerHeartbeater matches the Heartbeater interface.
var _ Heartbeater = (*localCheckerHeartbeater)(nil)

// LocalCheckerHeartbeaterConfig structure describing the localCheckerHeartbeater configuration.
type LocalCheckerHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	LocalCheckDelay time.Duration `validate:"required,gt=0"`
}

type localCheckerHeartbeater struct {
	*heartbeaterBase

	cfg             LocalCheckerHeartbeaterConfig
	lastChecksCount int64
}

// NewLocalCheckerHeartbeater is a function that creates a new localCheckerHeartbeater.
func NewLocalCheckerHeartbeater(cfg LocalCheckerHeartbeaterConfig, base *heartbeaterBase) (*localCheckerHeartbeater, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("local checker heartbeater configuration error: %w", err)
	}

	return &localCheckerHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

// Check is a function that checks that the local checker checks triggers and the number of triggers is not constant.
func (heartbeater *localCheckerHeartbeater) Check() (State, error) {
	triggersCount, err := heartbeater.database.GetTriggersToCheckCount(localClusterKey)
	if err != nil {
		return StateError, err
	}

	checksCount, err := heartbeater.database.GetChecksUpdatesCount()
	if err != nil {
		return StateError, err
	}

	now := heartbeater.clock.NowUTC()
	if heartbeater.lastChecksCount != checksCount || triggersCount == 0 {
		heartbeater.lastChecksCount = checksCount
		heartbeater.lastSuccessfulCheck = now
		return StateOK, nil
	}

	if now.Sub(heartbeater.lastSuccessfulCheck) > heartbeater.cfg.LocalCheckDelay {
		return StateError, nil
	}

	return StateOK, nil
}

// NeedTurnOffNotifier is a function that checks to see if the notifier needs to be turned off.
func (heartbeater localCheckerHeartbeater) NeedTurnOffNotifier() bool {
	return heartbeater.cfg.NeedTurnOffNotifier
}

// Type is a function that returns the current heartbeat type.
func (localCheckerHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatTypeNotSet
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater localCheckerHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
