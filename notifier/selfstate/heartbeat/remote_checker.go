package heartbeat

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

var (
	remoteClusterKey = moira.DefaultGraphiteRemoteCluster

	// Verify that remoteCheckerHeartbeater matches the Heartbeater interface.
	_ Heartbeater = (*remoteCheckerHeartbeater)(nil)
)

// RemoteCheckerHeartbeaterConfig structure describing the remoteCheckerHeartbeater configuration.
type RemoteCheckerHeartbeaterConfig struct {
	HeartbeaterBaseConfig

	RemoteCheckDelay time.Duration `validate:"required_if=Enabled true,gte=0"`
}

type remoteCheckerHeartbeater struct {
	*heartbeaterBase

	cfg                   RemoteCheckerHeartbeaterConfig
	lastRemoteChecksCount int64
}

// NewRemoteCheckerHeartbeater is a function that creates a new remoteCheckerHeartbeater.
func NewRemoteCheckerHeartbeater(cfg RemoteCheckerHeartbeaterConfig, base *heartbeaterBase) (*remoteCheckerHeartbeater, error) {
	if err := moira.ValidateStruct(cfg); err != nil {
		return nil, fmt.Errorf("remote checker heartbeater configuration error: %w", err)
	}

	return &remoteCheckerHeartbeater{
		heartbeaterBase: base,
		cfg:             cfg,
	}, nil
}

// Check is a function that checks that the remote checker checks triggers and the number of triggers is not constant.
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

// Type is a function that returns the current heartbeat type.
func (remoteCheckerHeartbeater) Type() datatypes.HeartbeatType {
	return datatypes.HeartbeatRemoteChecker
}

// AlertSettings is a function that returns the current settings for alerts.
func (heartbeater remoteCheckerHeartbeater) AlertSettings() AlertConfig {
	return heartbeater.cfg.AlertCfg
}
