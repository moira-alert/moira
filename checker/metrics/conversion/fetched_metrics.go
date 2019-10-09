package conversion

import (
	metricSource "github.com/moira-alert/moira/metric_source"
)

const defaultStep int64 = 60

// FetchedTargetMetrics represents different metrics within one target.
type FetchedTargetMetrics []metricSource.MetricData

// NewFetchedTargetMetrics is a constructor function for patternMetrics.
func NewFetchedTargetMetrics(source []metricSource.MetricData) FetchedTargetMetrics {
	result := NewFetchedTargetMetricsWithCapacity(len(source))
	for _, metric := range source {
		result = append(result, metric)
	}
	return result
}

// NewFetchedTargetMetricsWithCapacity is a constructor function for patternMetrics.
func NewFetchedTargetMetricsWithCapacity(capacity int) FetchedTargetMetrics {
	return make(FetchedTargetMetrics, 0, capacity)
}

// CleanWildcards is a function that removes all wildcarded metrics.
func (m FetchedTargetMetrics) CleanWildcards() FetchedTargetMetrics {
	result := NewFetchedTargetMetricsWithCapacity(len(m))
	for _, metric := range m {
		if !metric.Wildcard {
			result = append(result, metric)
		}
	}
	return result
}

// Deduplicate is a function that checks if FetchedPatternMetrics have a two or more metrics with
// the same name and returns new FetchedPatternMetrics without duplicates and slice of duplicated metrics names.
func (m FetchedTargetMetrics) Deduplicate() (FetchedTargetMetrics, []string) {
	deduplicated := NewFetchedTargetMetricsWithCapacity(len(m))
	collectedNames := make(setHelper, len(m))
	var duplicates []string
	for _, metric := range m {
		if collectedNames[metric.Name] {
			duplicates = append(duplicates, metric.Name)
		} else {
			deduplicated = append(deduplicated, metric)
		}
		collectedNames[metric.Name] = true
	}
	return deduplicated, duplicates
}

// FetchedMetrics represent collections of metrics associated with target name
// There is a map where keys are target names and values are maps of metrics with metric names as keys.
type FetchedMetrics map[string]FetchedTargetMetrics

// NewFetchedMetrics is a constructor function that creates FetchedMetrics from source metrics map.
func NewFetchedMetrics(source map[string][]metricSource.MetricData) FetchedMetrics {
	result := NewFetchedMetricsWithCapacity(len(source))
	for targetName, metrics := range source {
		result[targetName] = NewFetchedTargetMetrics(metrics)
	}
	return result
}

// NewFetchedMetricsWithCapacity is a constructor function that creates FetchedMetrics with initialized empty fields.
func NewFetchedMetricsWithCapacity(capacity int) FetchedMetrics {
	return make(FetchedMetrics, capacity)
}
