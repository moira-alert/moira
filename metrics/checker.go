package metrics

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	MetricsBySource        map[moira.ClusterKey]*CheckMetrics
	MetricEventsChannelLen Histogram
	UnusedTriggersCount    Histogram
	MetricEventsHandleTime Timer
}

// GetCheckMetrics return check metrics dependent on given trigger type
func (metrics *CheckerMetrics) GetCheckMetrics(trigger *moira.Trigger) (*CheckMetrics, error) {
	return metrics.GetCheckMetricsBySource(moira.MakeClusterKey(trigger.TriggerSource, moira.DefaultCluster))
}

// GetCheckMetricsBySource return check metrics dependent on given trigger type
func (metrics *CheckerMetrics) GetCheckMetricsBySource(clusterKey moira.ClusterKey) (*CheckMetrics, error) {
	if checkMetrics, ok := metrics.MetricsBySource[clusterKey]; ok {
		return checkMetrics, nil
	}

	return nil, fmt.Errorf("unknown cluster with key `%s`", clusterKey.String())
}

// CheckMetrics is a collection of metrics for trigger checks
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}

// ConfigureCheckerMetrics is checker metrics configurator
func ConfigureCheckerMetrics(registry Registry, sources []moira.ClusterKey) *CheckerMetrics {
	metrics := &CheckerMetrics{
		MetricsBySource:        make(map[moira.ClusterKey]*CheckMetrics),
		MetricEventsChannelLen: registry.NewHistogram("metricEvents"),
		MetricEventsHandleTime: registry.NewTimer("metricEventsHandle"),
		UnusedTriggersCount:    registry.NewHistogram("triggers", "unused"),
	}
	for _, clusterKey := range sources {
		metrics.MetricsBySource[clusterKey] = configureCheckMetrics(registry, clusterKey)
	}
	return metrics
}

func configureCheckMetrics(registry Registry, clusterKey moira.ClusterKey) *CheckMetrics {
	source, id := clusterKey.TriggerSource.String(), clusterKey.ClusterId.String()
	return &CheckMetrics{
		CheckError:           registry.NewMeter(source, id, "errors", "check"),
		HandleError:          registry.NewMeter(source, id, "errors", "handle"),
		TriggersCheckTime:    registry.NewTimer(source, id, "triggers"),
		TriggersToCheckCount: registry.NewHistogram(source, id, "triggersToCheck"),
	}
}
