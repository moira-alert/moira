package heartbeat

import (
	"time"

	"github.com/moira-alert/moira"
)

type State string

const (
	StateOK    State = "heartbeat_state_ok"
	StateError State = "heartbeat_state_error"
)

func (lastState State) IsDegradated(newState State) bool {
	return lastState == StateOK && newState == StateError
}

func (lastState State) IsRecovered(newState State) bool {
	return lastState == StateError && newState == StateOK
}

// Heartbeater is the interface for simplified events verification.
type Heartbeater interface {
	Check() (State, error)
	NeedTurnOffNotifier() bool
	NeedToCheckOthers() bool
	AlertSettings() AlertConfig
	Type() moira.EmergencyContactType
}

type HeartbeaterBaseConfig struct {
	Enabled             bool
	NeedTurnOffNotifier bool
	NeedToCheckOthers   bool

	AlertCfg AlertConfig `validate:"required_if=Enabled true"`
}

type AlertConfig struct {
	Name string `validate:"required_if=Enabled true"`
	Desc string
}

// heartbeat basic structure for Heartbeater.
type heartbeaterBase struct {
	logger   moira.Logger
	database moira.Database
	clock    moira.Clock

	lastSuccessfulCheck time.Time
}

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
