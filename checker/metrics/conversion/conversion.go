package conversion

import (
	"sort"

	metricSource "github.com/moira-alert/moira/metric_source"
)

// isOneMetricMap is a function that checks that map have only one metric and if so returns that metric key.
func isOneMetricMap(metrics map[string]metricSource.MetricData) (bool, string) {
	if len(metrics) == 1 {
		for metricName := range metrics {
			return true, metricName
		}
	}
	return false, ""
}

// MetricName is a function that returns a metric name from random metric in MetricsToCheck.
// Should be used with care if MetricsToCheck have metrics with different names.
func MetricName(metrics map[string]metricSource.MetricData) string {
	if len(metrics) == 0 {
		return ""
	}
	var metricNames []string
	for _, metric := range metrics {
		metricNames = append(metricNames, metric.Name)
	}
	sort.Strings(metricNames)
	return metricNames[0]
}

// GetRelations is a function that returns a map with relation between target name and metric
// name for this target.
func GetRelations(metrics map[string]metricSource.MetricData) map[string]string {
	result := make(map[string]string, len(metrics))
	for targetName, metric := range metrics {
		result[targetName] = metric.Name
	}
	return result
}

// Merge is a function that merges two metricSource.MetricData maps and returns a map
// where represented elements from both maps.
func Merge(metrics map[string]metricSource.MetricData, other map[string]metricSource.MetricData) map[string]metricSource.MetricData {
	result := make(map[string]metricSource.MetricData, len(metrics)+len(other))
	for k, v := range metrics {
		result[k] = v
	}
	for k, v := range other {
		result[k] = v
	}
	return result
}

// HasOnlyWildcards is a function that checks that metrics are only wildcards.
func HasOnlyWildcards(metrics map[string][]metricSource.MetricData) bool {
	for _, patternMetrics := range metrics {
		for _, timeSeries := range patternMetrics {
			if !timeSeries.Wildcard {
				return false
			}
		}
	}
	return true
}
