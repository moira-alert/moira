package graphite

import "github.com/moira-alert/moira"

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	MoiraMetrics           *CheckMetrics
	RemoteMetrics          *CheckMetrics
	MetricEventsChannelLen Histogram
	UnusedTriggersCount    Histogram
	MetricEventsHandleTime Timer
}

// GetCheckMetrics return check metrics dependent on given trigger type
func (metrics *CheckerMetrics) GetCheckMetrics(trigger *moira.Trigger) *CheckMetrics {
	if trigger.IsRemote {
		return metrics.RemoteMetrics
	}
	return metrics.MoiraMetrics
}

// CheckMetrics is a collection of metrics for trigger checks
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}
