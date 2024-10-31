package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/datatypes"
)

// State characterises the state of the heartbeat.
type State string

const (
	StateOK    State = "heartbeat_state_ok"
	StateError State = "heartbeat_state_error"
)

// IsDegraded checks if the condition has degraded.
func (lastState State) IsDegraded(newState State) bool {
	return lastState == StateOK && newState == StateError
}

// IsRecovered checks if the condition has recovered.
func (lastState State) IsRecovered(newState State) bool {
	return lastState == StateError && newState == StateOK
}

// Heartbeater is the interface for simplified events verification.
type Heartbeater interface {
	Check() (State, error)
	NeedTurnOffNotifier() bool
	AlertSettings() AlertConfig
	Type() datatypes.HeartbeatType
}

// HeartbeaterBaseConfig contains common fields for all heartbeaters.
type HeartbeaterBaseConfig struct {
	Enabled             bool
	NeedTurnOffNotifier bool
	NeedToCheckOthers   bool

	AlertCfg AlertConfig `validate:"required_if=Enabled true"`
}

// AlertConfig contains the configuration of the alerts that heartbeater sends out.
type AlertConfig struct {
	Name string `validate:"required_if=Enabled true"`
	Desc string
}

// HeartbeatBase is basic structure for Heartbeater.
type heartbeaterBase struct {
	logger   moira.Logger
	database moira.Database
	clock    moira.Clock

	lastSuccessfulCheck time.Time
}

// NewHeartbeaterBase function that creates a base for heartbeater.
func NewHeartbeaterBase(
	logger moira.Logger,
	database moira.Database,
	clock moira.Clock,
) *heartbeaterBase {
	return &heartbeaterBase{
		logger:   logger,
		database: database,
		clock:    clock,

		lastSuccessfulCheck: clock.NowUTC(),
	}
}
