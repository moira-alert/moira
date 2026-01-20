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
	// TODO: Remove after v2.17.0 release
	// Deprecated: use counter instead
	CheckError        Meter
	CheckErrorCounter Counter
	// TODO: Remove after v2.17.0 release
	// Deprecated: use counter instead
	HandleError          Meter
	HandleErrorCounter   Counter
	TriggersCheckTime    Timer
	TriggersToCheckCount Histogram
}

// ConfigureCheckerMetrics is checker metrics configurator.
func ConfigureCheckerMetrics(registry Registry, attributedRegistry MetricRegistry, sources []moira.ClusterKey, settings Settings) (*CheckerMetrics, error) {
	const metricEventsChannelLenMetric string = "metric.events.count"
	metricEventsChannelLen, err := attributedRegistry.NewHistogram(metricEventsChannelLenMetric, settings.GetHistogramBacketOr(metricEventsChannelLenMetric, DefaultHistogramBackets))
	if err != nil {
		return nil, err
	}

	const metricEventsHandleTimeMetric string = "metric.events.handle_time"
	metricEventsHandleTime, err := attributedRegistry.NewTimer(metricEventsHandleTimeMetric, settings.GetTimerBacketOr(metricEventsHandleTimeMetric, DefaultTimerBackets))
	if err != nil {
		return nil, err
	}

	const unusedTriggersCountMetric string = "triggers.unused.count"
	unusedTriggersCount, err := attributedRegistry.NewHistogram(unusedTriggersCountMetric, DefaultHistogramBackets)
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
		checkMetrics, err := configureCheckMetrics(registry, attributedRegistry, clusterKey, settings)
		if err != nil {
			return nil, err
		}

		metrics.MetricsBySource[clusterKey] = checkMetrics
	}

	return metrics, nil
}

func configureCheckMetrics(registry Registry, attributedRegistry MetricRegistry, clusterKey moira.ClusterKey, settings Settings) (*CheckMetrics, error) {
	source, id := clusterKey.TriggerSource.String(), clusterKey.ClusterId.String()
	metricRegistrySourced := attributedRegistry.WithAttributes(Attributes{
		Attribute{"metric.source.name", source},
		Attribute{"metric.source.id", id},
	})

	checkError, err := metricRegistrySourced.NewCounter("triggers.check.errors.count")
	if err != nil {
		return nil, err
	}

	handleError, err := metricRegistrySourced.NewCounter("triggers.handle.errors.count")
	if err != nil {
		return nil, err
	}

	const triggersCheckTimeMetric string = "triggers.check.time"
	triggersCheckTime, err := metricRegistrySourced.NewTimer(triggersCheckTimeMetric, settings.GetTimerBacketOr(triggersCheckTimeMetric, DefaultTimerBackets))
	if err != nil {
		return nil, err
	}

	const triggersToCheckCountMetric string = "triggers.to_check.count"
	triggersToCheckCount, err := metricRegistrySourced.NewHistogram("triggers.to_check.count", settings.GetHistogramBacketOr(triggersToCheckCountMetric, DefaultHistogramBackets))
	if err != nil {
		return nil, err
	}

	return &CheckMetrics{
		// Deprecated: only triggers.check.errors.count metric of metricRegistrySourced should be used.
		CheckError:        registry.NewMeter(source, id, "errors", "check"),
		CheckErrorCounter: checkError,
		// Deprecated: only triggers.handle.errors.count metric of metricRegistrySourced should be used.
		HandleError:        registry.NewMeter(source, id, "errors", "handle"),
		HandleErrorCounter: handleError,
		// Deprecated: only triggers.check.time metric of metricRegistrySourced should be used.
		TriggersCheckTime: NewCompositeTimer(registry.NewTimer(source, id, "triggers"), triggersCheckTime),
		// Deprecated: only triggers.to_check_count metric of metricRegistrySourced should be used.
		TriggersToCheckCount: NewCompositeHistogram(registry.NewHistogram(source, id, "triggersToCheck"), triggersToCheckCount),
	}, nil
}
