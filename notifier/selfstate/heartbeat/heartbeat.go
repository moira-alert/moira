package heartbeat

import (
	"github.com/moira-alert/moira"
)

const templateMoreThanMessage = "%s more than %ds. Send message."

// Heartbeater is the interface for simplified events verification.
type Heartbeater interface {
	Check(int64) (int64, bool, error)
	NeedTurnOffNotifier() bool
	NeedToCheckOthers() bool
	GetErrorMessage() string
}

// heartbeat basic structure for Heartbeater
type heartbeat struct {
	logger   moira.Logger
	database moira.Database

	delay, lastSuccessfulCheck int64
}
