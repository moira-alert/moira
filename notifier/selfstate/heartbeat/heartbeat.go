package heartbeat

import (
	"github.com/moira-alert/moira"
)

// Heartbeater is the interface for simplified events verification.
type Heartbeater interface {
	Check(int64) (int64, bool, error)
	NeedTurnOffNotifier() bool
	NeedToCheckOthers() bool
	GetErrorMessage() string
	GetCheckTags() CheckTags
}

// CheckTags represents a tag collection.
type CheckTags []string

// heartbeat basic structure for Heartbeater.
type heartbeat struct {
	logger   moira.Logger
	database moira.Database

	checkTags                  []string
	delay, lastSuccessfulCheck int64
}
