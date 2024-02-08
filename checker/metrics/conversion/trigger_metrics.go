package conversion

import (
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// TriggerTargetMetrics is a map that contains metrics of one target. Keys of this map
// are metric names. This map have a methods that helps to prepare metrics for check.
type TriggerTargetMetrics map[string]metricSource.MetricData

// newTriggerTargetMetricsWithCapacity is a constructor function for TriggerTargetMetrics that creates
// a new map with given capacity.
func newTriggerTargetMetricsWithCapacity(capacity int) TriggerTargetMetrics {
	return make(TriggerTargetMetrics, capacity)
}

// NewTriggerTargetMetrics is a constructor function for TriggerTargetMetrics that creates
// a new empty map.
func NewTriggerTargetMetrics(source FetchedTargetMetrics) TriggerTargetMetrics {
	result := newTriggerTargetMetricsWithCapacity(len(source))
	for _, m := range source {
		result[m.Name] = m
	}
	return result
}

// Populate is a function that takes the list of metric names that first appeared and
// adds metrics with this names and empty values.
func (m TriggerTargetMetrics) Populate(lastMetrics set[string], from, to int64) TriggerTargetMetrics {
	result := newTriggerTargetMetricsWithCapacity(len(m))

	var firstMetric metricSource.MetricData

	for _, metric := range m {
		firstMetric = metric
		break
	}

	for metricName := range lastMetrics {
		metric, ok := m[metricName]
		if !ok {
			step := defaultStep
			if len(m) > 0 && firstMetric.StepTime != 0 {
				step = firstMetric.StepTime
			}
			metric = *metricSource.MakeEmptyMetricData(metricName, step, from, to)
		}
		result[metricName] = metric
	}
	return result
}

// TriggerMetrics is a map of TriggerTargetMetrics that represents all metrics within trigger.
type TriggerMetrics map[string]TriggerTargetMetrics

// NewTriggerMetricsWithCapacity is a constructor function that creates TriggerMetrics with given capacity.
func NewTriggerMetricsWithCapacity(capacity int) TriggerMetrics {
	return make(TriggerMetrics, capacity)
}

// Populate is a function that takes TriggerMetrics and populate targets
// that is missing metrics that appear in another targets except the targets that have
// only alone metrics.
func (triggerMetrics TriggerMetrics) Populate(lastMetrics map[string]moira.MetricState, declaredAloneMetrics map[string]bool, from int64, to int64) TriggerMetrics {
	// This one have all metrics that should be in final TriggerMetrics.
	// This structure filled with metrics from last check,
	// current received metrics alone metrics from last check.
	allMetrics := make(map[string]set[string], len(triggerMetrics))

	for metricName, metricState := range lastMetrics {
		for targetName := range metricState.Values {
			if _, ok := allMetrics[targetName]; !ok {
				allMetrics[targetName] = make(set[string])
			}

			allMetrics[targetName].insert(metricName)
		}
	}

	for targetName, metrics := range triggerMetrics {
		if _, ok := allMetrics[targetName]; !ok {
			allMetrics[targetName] = make(set[string])
		}

		for metricName := range metrics {
			allMetrics[targetName].insert(metricName)
		}
	}

	diff := triggerMetrics.FindMissingMetrics(newSet(declaredAloneMetrics))

	for targetName, metrics := range diff {
		for metricName := range metrics {
			allMetrics[targetName].insert(metricName)
		}
	}

	result := NewTriggerMetricsWithCapacity(len(allMetrics))
	for targetName, metrics := range allMetrics {
		// // We do not populate metrics
		// if declaredAloneMetrics[targetName] {
		// 	continue
		// }
		targetMetrics, ok := triggerMetrics[targetName]
		if !ok {
			targetMetrics = newTriggerTargetMetricsWithCapacity(len(metrics))
		}
		targetMetrics = targetMetrics.Populate(metrics, from, to)
		result[targetName] = targetMetrics
	}

	return result
}

// FilterAloneMetrics is a function that remove alone metrics targets from TriggerMetrics
// and return this metrics in format map[targetName]MetricData.
// We split targets that declared as targets with alone metrics
// from targets with multiple metrics.
//
//	For example, we have a targets with metrics:
//	{
//		"t1": {"m1": {metrics}, "m2": {metrics}, "m3": {metrics}},
//		"t2": {"m1": {metrics}, "m2": {metrics}, "m3": {metrics}},
//		"t3": {"m4": {metrics}},
//	}
//
// and declared alone metrics
//
//	{"t3": true}
//
// These methods will return
//
//	{
//		"t1": {"m1", "m2", "m3"},
//		"t2": {"m1", "m2", "m3"},
//	}
//
// and
//
//	{
//	"t3": {metrics},
//	}
func (triggerMetrics TriggerMetrics) FilterAloneMetrics(declaredAloneMetrics map[string]bool) (TriggerMetrics, AloneMetrics, error) {
	if len(declaredAloneMetrics) == 0 {
		return triggerMetrics, NewAloneMetricsWithCapacity(0), nil
	}

	metricCountUpperBound := len(triggerMetrics)
	result := NewTriggerMetricsWithCapacity(metricCountUpperBound)
	aloneMetrics := NewAloneMetricsWithCapacity(metricCountUpperBound)

	errorBuilder := newErrUnexpectedAloneMetricBuilder()
	errorBuilder.setDeclared(declaredAloneMetrics)

	for targetName, targetMetrics := range triggerMetrics {
		if !declaredAloneMetrics[targetName] {
			result[targetName] = triggerMetrics[targetName]
			continue
		}

		oneMetricMap, metricName := isOneMetricMap(targetMetrics)
		if !oneMetricMap {
			if len(targetMetrics) == 0 {
				continue
			}
			errorBuilder.addUnexpected(targetName, targetMetrics)
			continue
		}

		aloneMetrics[targetName] = targetMetrics[metricName]
	}

	if err := errorBuilder.build(); err != nil {
		return TriggerMetrics{}, AloneMetrics{}, err
	}

	return result, aloneMetrics, nil
}

// FindMissingMetrics is a function that returns a map of target names with metric names that are absent in
// current target but appear in another targets.
func (triggerMetrics TriggerMetrics) FindMissingMetrics(declaredAloneMetrics set[string]) map[string]set[string] {
	result := make(map[string]set[string])

	if len(triggerMetrics) == 0 {
		return result
	}

	fullMetrics := make(set[string])

	for targetName, targetMetrics := range triggerMetrics {
		if declaredAloneMetrics.contains(targetName) {
			continue
		}
		currentMetrics := newSetFromTriggerTargetMetrics(targetMetrics)
		fullMetrics = fullMetrics.union(currentMetrics)
	}

	for targetName, targetMetrics := range triggerMetrics {
		metricsSet := newSetFromTriggerTargetMetrics(targetMetrics)
		if declaredAloneMetrics.contains(targetName) {
			continue
		}
		diff := metricsSet.diff(fullMetrics)
		if len(diff) > 0 {
			result[targetName] = diff
		}
	}
	return result
}

// ConvertForCheck is a function that converts TriggerMetrics with structure
// map[TargetName]map[MetricName]MetricData to ConvertedTriggerMetrics
// with structure map[MetricName]map[TargetName]MetricData.
func (m TriggerMetrics) ConvertForCheck() map[string]map[string]metricSource.MetricData {
	result := make(map[string]map[string]metricSource.MetricData)
	for targetName, targetMetrics := range m {
		for metricName := range targetMetrics {
			if _, ok := result[metricName]; !ok {
				result[metricName] = make(map[string]metricSource.MetricData, len(m))
			}
			result[metricName][targetName] = m[targetName][metricName]
		}
	}
	return result
}
