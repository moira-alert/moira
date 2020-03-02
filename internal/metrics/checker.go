package metrics

import (
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	LocalMetrics           *CheckMetrics
	RemoteMetrics          *CheckMetrics
	MetricEventsChannelLen Histogram
	UnusedTriggersCount    Histogram
	MetricEventsHandleTime Timer
}

// GetCheckMetrics return check metrics dependent on given trigger type
func (metrics *CheckerMetrics) GetCheckMetrics(trigger *moira2.Trigger) *CheckMetrics {
	if trigger.IsRemote {
		return metrics.RemoteMetrics
	}
	return metrics.LocalMetrics
}

// CheckMetrics is a collection of metrics for trigger checks
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}

// ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(registry Registry, remoteEnabled bool) *CheckerMetrics {
	m := &CheckerMetrics{
		LocalMetrics:           configureCheckMetrics(registry, "local"),
		MetricEventsChannelLen: registry.NewHistogram("metricEvents"),
		MetricEventsHandleTime: registry.NewTimer("metricEventsHandle"),
		UnusedTriggersCount:    registry.NewHistogram("triggers", "unused"),
	}
	if remoteEnabled {
		m.RemoteMetrics = configureCheckMetrics(registry, "remote")
	}
	return m
}

func configureCheckMetrics(registry Registry, prefix string) *CheckMetrics {
	return &CheckMetrics{
		CheckError:           registry.NewMeter(prefix, "errors", "check"),
		HandleError:          registry.NewMeter(prefix, "errors", "handle"),
		TriggersCheckTime:    registry.NewTimer(prefix, "triggers"),
		TriggersToCheckCount: registry.NewHistogram(prefix, "triggersToCheck"),
	}
}
