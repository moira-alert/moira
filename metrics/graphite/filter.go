package graphite

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// FilterMetrics is a collection of metrics used in filter
type FilterMetrics struct {
	RuntimeMetricsRegistry  *goMetrics.Registry
	TotalMetricsReceived    Counter
	ValidMetricsReceived    Counter
	MatchingMetricsReceived Counter
	MatchingTimer           Timer
	SavingTimer             Timer
	BuildTreeTimer          Timer
	MetricChannelLen        Histogram
}
