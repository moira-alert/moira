package metrics

import "github.com/moira-alert/moira"

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	LocalMetrics           *CheckMetrics
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
func ConfigureCheckerMetrics(registry Registry, prefix string, remoteEnabled bool) *CheckerMetrics {
	m := &CheckerMetrics{
		LocalMetrics:           configureCheckMetrics(registry, metricNameWithPrefix(prefix, "local")),
		MetricEventsChannelLen: registry.NewHistogram(metricNameWithPrefix(prefix, "metricEvents")),
		MetricEventsHandleTime: registry.NewTimer(metricNameWithPrefix(prefix, "metricEventsHandle")),
		UnusedTriggersCount:    registry.NewHistogram(metricNameWithPrefix(prefix, "triggers.unused")),
	}
	if remoteEnabled {
		m.RemoteMetrics = configureCheckMetrics(registry, metricNameWithPrefix(prefix, "remote"))
	}
	return m
}

func configureCheckMetrics(registry Registry, prefix string) *CheckMetrics {
	return &CheckMetrics{
		CheckError:           registry.NewMeter(metricNameWithPrefix(prefix, "errors.check")),
		HandleError:          registry.NewMeter(metricNameWithPrefix(prefix, "errors.handle")),
		TriggersCheckTime:    registry.NewTimer(metricNameWithPrefix(prefix, "triggers")),
		TriggersToCheckCount: registry.NewHistogram(metricNameWithPrefix(prefix, "triggersToCheck")),
	}
}
