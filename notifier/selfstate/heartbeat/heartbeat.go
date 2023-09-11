package heartbeat

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

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
	metrics  *metrics.HeartBeatMetrics

	delay, lastSuccessfulCheck int64
}

type Metrics interface {
	GetMeter() metrics.Meter
}
