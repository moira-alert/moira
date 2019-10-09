package local

import (
	metricSource "github.com/moira-alert/moira/metric_source"
)

// FetchResult is implementation of metric_source.FetchResult interface,
// which represent fetching result from moira data source in moira format
type FetchResult struct {
	MetricsData []metricSource.MetricData
	Patterns    []string
	Metrics     []string
}

// CreateEmptyFetchResult just creates FetchResult with initialized empty fields
func CreateEmptyFetchResult() *FetchResult {
	return &FetchResult{
		MetricsData: make([]metricSource.MetricData, 0),
		Patterns:    make([]string, 0),
		Metrics:     make([]string, 0),
	}
}

// GetMetricsData return all metrics data from fetch result
func (fetchResult *FetchResult) GetMetricsData() []metricSource.MetricData {
	return fetchResult.MetricsData
}

// GetPatterns return all patterns which contains in evaluated graphite target
func (fetchResult *FetchResult) GetPatterns() ([]string, error) {
	return fetchResult.Patterns, nil
}

// GetPatternMetrics return all metrics which match to evaluated graphite target patterns
func (fetchResult *FetchResult) GetPatternMetrics() ([]string, error) {
	return fetchResult.Metrics, nil
}
