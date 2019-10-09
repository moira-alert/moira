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
func (m TriggerTargetMetrics) Populate(lastMetrics map[string]bool, from, to int64) TriggerTargetMetrics {
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
func (m TriggerMetrics) Populate(lastCheck moira.CheckData, from int64, to int64) TriggerMetrics {
	allMetrics := make(map[string]map[string]bool, len(m))
	lastAloneMetrics := make(map[string]bool, len(lastCheck.MetricsToTargetRelation))

	for targetName, metricName := range lastCheck.MetricsToTargetRelation {
		allMetrics[targetName] = map[string]bool{metricName: true}
		lastAloneMetrics[metricName] = true
	}

	for metricName, metricState := range lastCheck.Metrics {
		if lastAloneMetrics[metricName] {
			continue
		}
		for targetName := range metricState.Values {
			if _, ok := lastCheck.MetricsToTargetRelation[targetName]; ok {
				continue
			}
			if _, ok := allMetrics[targetName]; !ok {
				allMetrics[targetName] = make(map[string]bool)
			}
			allMetrics[targetName][metricName] = true
		}
	}
	for targetName, metrics := range m {
		for metricName := range metrics {
			if _, ok := allMetrics[targetName]; !ok {
				allMetrics[targetName] = make(map[string]bool)
			}
			allMetrics[targetName][metricName] = true
		}
	}

	diff := m.Diff()

	for targetName, metrics := range diff {
		for metricName := range metrics {
			allMetrics[targetName][metricName] = true
		}
	}

	result := NewTriggerMetricsWithCapacity(len(allMetrics))
	for targetName, metrics := range allMetrics {
		targetMetrics, ok := m[targetName]
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
func (m TriggerMetrics) FilterAloneMetrics() (TriggerMetrics, map[string]metricSource.MetricData) {
	result := NewTriggerMetricsWithCapacity(len(m))
	aloneMetrics := make(map[string]metricSource.MetricData)

	for targetName, targetMetrics := range m {
		if oneMetricMap, metricName := isOneMetricMap(targetMetrics); oneMetricMap {
			aloneMetrics[targetName] = targetMetrics[metricName]
			continue
		}
		result[targetName] = m[targetName]
	}
	return result, aloneMetrics
}

// Diff is a function that returns a map of target names with metric names that are absent in
// current target but appear in another targets.
func (m TriggerMetrics) Diff() map[string]map[string]bool {
	result := make(map[string]map[string]bool)

	if len(m) == 0 {
		return result
	}

	fullMetrics := make(setHelper)

	for _, targetMetrics := range m {
		if oneMetricTarget, _ := isOneMetricMap(targetMetrics); oneMetricTarget {
			continue
		}
		currentMetrics := newSetHelperFromTriggerTargetMetrics(targetMetrics)
		fullMetrics = fullMetrics.union(currentMetrics)
	}

	for targetName, targetMetrics := range m {
		metricsSet := newSetHelperFromTriggerTargetMetrics(targetMetrics)
		if oneMetricTarget, _ := isOneMetricMap(targetMetrics); oneMetricTarget {
			continue
		}
		diff := metricsSet.diff(fullMetrics)
		if len(diff) > 0 {
			result[targetName] = diff
		}
	}
	return result
}

// multiMetricsTarget is a function that finds any first target with
// amount of metrics greater than one and returns set with names of this metrics.
func (m TriggerMetrics) multiMetricsTarget() (string, setHelper) {
	commonMetrics := make(setHelper)
	for targetName, metrics := range m {
		if len(metrics) > 1 {
			for metricName := range metrics {
				commonMetrics[metricName] = true
			}
			return targetName, commonMetrics
		}
	}
	return "", nil
}

// ConvertForCheck is a function that converts TriggerMetrics with structure
// map[TargetName]map[MetricName]MetricData to ConvertedTriggerMetrics
// with structure map[MetricName]map[TargetName]MetricData and fill with
// duplicated metrics targets that have only one metric. Second return value is
// a map with names of targets that had only one metric as key and original metric name as value.
func (m TriggerMetrics) ConvertForCheck() map[string]map[string]metricSource.MetricData {
	result := make(map[string]map[string]metricSource.MetricData)
	_, commonMetrics := m.multiMetricsTarget()

	hasAtLeastOneMultiMetricsTarget := commonMetrics != nil

	if !hasAtLeastOneMultiMetricsTarget && len(m) <= 1 {
		return result
	}

	for targetName, targetMetrics := range m {
		oneMetricTarget, oneMetricName := isOneMetricMap(targetMetrics)

		for metricName := range commonMetrics {
			if _, ok := result[metricName]; !ok {
				result[metricName] = make(map[string]metricSource.MetricData, len(m))
			}

			if oneMetricTarget {
				result[metricName][targetName] = m[targetName][oneMetricName]
				continue
			}

			result[metricName][targetName] = m[targetName][metricName]
		}
	}
	return result
}
