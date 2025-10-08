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
func ConfigureCheckerMetrics(registry Registry, attributedRegistry MetricRegistry, sources []moira.ClusterKey) (*CheckerMetrics, error) {
	metricEventsChannelLen, err := attributedRegistry.NewHistogram("metricEvents")
	if err != nil {
		return nil, err
	}

	metricEventsHandleTime, err := attributedRegistry.NewTimer("metricEventsHandle")
	if err != nil {
		return nil, err
	}

	unusedTriggersCount, err := attributedRegistry.NewHistogram("unused_triggers")
	if err != nil {
		return nil, err
	}

	metrics := &CheckerMetrics{
		MetricsBySource:        make(map[moira.ClusterKey]*CheckMetrics),
		MetricEventsChannelLen: NewCompositeHistogram(registry.NewHistogram("metricEvents"), metricEventsChannelLen),
		MetricEventsHandleTime: NewCompositeTimer(registry.NewTimer("metricEventsHandle"), metricEventsHandleTime),
		UnusedTriggersCount:    NewCompositeHistogram(registry.NewHistogram("triggers", "unused"), unusedTriggersCount),
	}

	for _, clusterKey := range sources {
		checkMetrics, err := configureCheckMetrics(registry, attributedRegistry, clusterKey)
		if err != nil {
			return nil, err
		}

		metrics.MetricsBySource[clusterKey] = checkMetrics
	}

	return metrics, nil
}

func configureCheckMetrics(registry Registry, attributedRegistry MetricRegistry, clusterKey moira.ClusterKey) (*CheckMetrics, error) {
	source, id := clusterKey.TriggerSource.String(), clusterKey.ClusterId.String()
	metricRegistrySourced := attributedRegistry.WithAttributes(Attributes{
		Attribute{"metric_source", source},
		Attribute{"metric_source_id", id},
	})

	checkError, err := metricRegistrySourced.NewGauge("triggers.check.errors.count")
	if err != nil {
		return nil, err
	}

	handleError, err := metricRegistrySourced.NewGauge("triggers.handle.errors.count")
	if err != nil {
		return nil, err
	}

	triggersCheckTime, err := metricRegistrySourced.NewTimer("triggers.check.time")
	if err != nil {
		return nil, err
	}

	triggersToCheckCount, err := metricRegistrySourced.NewHistogram("triggers.to_check.count")
	if err != nil {
		return nil, err
	}

	return &CheckMetrics{
		CheckError:           NewCompositeMeter(registry.NewMeter(source, id, "errors", "check"), checkError),
		HandleError:          NewCompositeMeter(registry.NewMeter(source, id, "errors", "handle"), handleError),
		TriggersCheckTime:    NewCompositeTimer(registry.NewTimer(source, id, "triggers"), triggersCheckTime),
		TriggersToCheckCount: NewCompositeHistogram(registry.NewHistogram(source, id, "triggersToCheck"), triggersToCheckCount),
	}, nil
}
