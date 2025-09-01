package metrics

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// CheckerMetrics is a collection of metrics used in checker.
type CheckerMetrics struct {
	MetricsBySource        map[moira.ClusterKey]*CheckMetrics
	MetricEventsChannelLen Histogram
	UnusedTriggersCount    Histogram
	MetricEventsHandleTime Timer
}

// GetCheckMetrics return check metrics dependent on given trigger type.
func (metrics *CheckerMetrics) GetCheckMetrics(trigger *moira.Trigger) (*CheckMetrics, error) {
	return metrics.GetCheckMetricsBySource(trigger.ClusterKey())
}

// GetCheckMetricsBySource return check metrics dependent on given trigger type.
func (metrics *CheckerMetrics) GetCheckMetricsBySource(clusterKey moira.ClusterKey) (*CheckMetrics, error) {
	if checkMetrics, ok := metrics.MetricsBySource[clusterKey]; ok {
		return checkMetrics, nil
	}

	return nil, fmt.Errorf("can't get check metrics: unknown cluster with key `%s`", clusterKey.String())
}

// CheckMetrics is a collection of metrics for trigger checks.
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}

// ConfigureCheckerMetrics is checker metrics configurator.
func ConfigureCheckerMetrics(registry Registry, attributedregistry MetricRegistry, sources []moira.ClusterKey) *CheckerMetrics {
	metrics := &CheckerMetrics{
		MetricsBySource:        make(map[moira.ClusterKey]*CheckMetrics),
		MetricEventsChannelLen: NewCompositeHistogram(registry.NewHistogram("metricEvents"), attributedregistry.NewHistogram("metricEvents")),
		MetricEventsHandleTime: NewCompositeTimer(registry.NewTimer("metricEventsHandle"), attributedregistry.NewTimer("metricEventsHandle")),
		UnusedTriggersCount:    NewCompositeHistogram(registry.NewHistogram("triggers", "unused"), attributedregistry.NewHistogram("triggers_unused")),
	}
	for _, clusterKey := range sources {
		metrics.MetricsBySource[clusterKey] = configureCheckMetrics(registry, attributedregistry, clusterKey)
	}

	return metrics
}

func configureCheckMetrics(registry Registry, attributedregistry MetricRegistry, clusterKey moira.ClusterKey) *CheckMetrics {
	source, id := clusterKey.TriggerSource.String(), clusterKey.ClusterId.String()
	attributedByClusterRegistry := attributedregistry.WithAttributes(Attributes{
		Attribute{"metric_source", source},
		Attribute{"metric_cluster_id", id},
	})

	return &CheckMetrics{
		CheckError:           NewCompositeMeter(registry.NewMeter(source, id, "errors", "check"), attributedByClusterRegistry.NewGauge("errors_check")),
		HandleError:          NewCompositeMeter(registry.NewMeter(source, id, "errors", "handle"), attributedByClusterRegistry.NewGauge("errors_handle")),
		TriggersCheckTime:    NewCompositeTimer(registry.NewTimer(source, id, "triggers"), attributedByClusterRegistry.NewTimer("triggers")),
		TriggersToCheckCount: NewCompositeHistogram(registry.NewHistogram(source, id, "triggersToCheck"), attributedByClusterRegistry.NewHistogram("triggersToCheck")),
	}
}
