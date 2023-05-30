package metrics

import "github.com/moira-alert/moira"

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	LocalMetrics           *CheckMetrics
	RemoteMetrics          *CheckMetrics
	VMSelectMetrics        *CheckMetrics
	MetricEventsChannelLen Histogram
	UnusedTriggersCount    Histogram
	MetricEventsHandleTime Timer
}

// GetCheckMetrics return check metrics dependent on given trigger type
func (metrics *CheckerMetrics) GetCheckMetrics(trigger *moira.Trigger) *CheckMetrics {
	/// if trigger.IsRemote {
	/// 	return metrics.RemoteMetrics
	/// }
	/// return metrics.LocalMetrics
	switch trigger.TriggerSource {
	case moira.GraphiteLocal:
		return metrics.LocalMetrics

	case moira.GraphiteRemote:
		return metrics.RemoteMetrics

	case moira.VMSelectRemote:
		return metrics.VMSelectMetrics

	default:
		return nil
	}
}

// CheckMetrics is a collection of metrics for trigger checks
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}

// ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(registry Registry, remoteEnabled, vmselectEnabled bool) *CheckerMetrics {
	m := &CheckerMetrics{
		LocalMetrics:           configureCheckMetrics(registry, "local"),
		MetricEventsChannelLen: registry.NewHistogram("metricEvents"),
		MetricEventsHandleTime: registry.NewTimer("metricEventsHandle"),
		UnusedTriggersCount:    registry.NewHistogram("triggers", "unused"),
	}
	if remoteEnabled {
		m.RemoteMetrics = configureCheckMetrics(registry, "remote")
	}
	if vmselectEnabled {
		m.VMSelectMetrics = configureCheckMetrics(registry, "vmselect")
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
