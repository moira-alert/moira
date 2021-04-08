package conversion

import (
	metricSource "github.com/moira-alert/moira/metric_source"
)

type AloneMetrics map[string]metricSource.MetricData

// NewAloneMetricsWithCapacity is a constructor function for AloneMetrics
func NewAloneMetricsWithCapacity(capacity int) AloneMetrics {
	return make(map[string]metricSource.MetricData, capacity)
}

// Populate is a method that tries to restore alone metrics that were in last check but absent in current check
// for example lastCheckMetricsToTargetRelation is:
//	{
//		"t2": "metric.name.1",
//		"t3": "metric.name.2",
//	}
// and current alone metrics are
//	{
//		"t2": metricSource.MetricData{Name: "metric.name.1"}
//	}
// result will be:
//	{
//		"t2": metricSource.MetricData{Name: "metric.name.1"},
//		"t3": metricSource.MetricData{Name: "metric.name.2"},
//	}
func (m AloneMetrics) Populate(lastCheckMetricsToTargetRelation map[string]string, declaredAloneMetrics map[string]bool, from, to int64) (AloneMetrics, error) {
	result := NewAloneMetricsWithCapacity(len(m))

	var firstMetric metricSource.MetricData

	// TODO(litleleprikon): check if it is ok to get step time from metric of neighbor target
	for _, metric := range m {
		firstMetric = metric
		break
	}

	for targetName := range declaredAloneMetrics {
		metricName, existInLastCheck := lastCheckMetricsToTargetRelation[targetName]
		metric, existInCurrentAloneMetrics := m[targetName]
		if !existInCurrentAloneMetrics && !existInLastCheck {
			return AloneMetrics{}, NewErrEmptyAloneMetricsTarget(targetName)
		}
		if !existInCurrentAloneMetrics {
			step := defaultStep
			if len(m) > 0 && firstMetric.StepTime != 0 {
				step = firstMetric.StepTime
			}
			metric = *metricSource.MakeEmptyMetricData(metricName, step, from, to)
		}
		result[targetName] = metric
	}

	return result, nil
}
