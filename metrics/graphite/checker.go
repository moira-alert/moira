package graphite

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	RuntimeMetricsRegistry    *goMetrics.Registry
	CheckError                Meter
	HandleError               Meter
	TriggersCheckTime         Timer
	TriggerCheckTime          TimerMap
	TriggersToCheckChannelLen Histogram
	MetricEventsChannelLen    Histogram
	MetricEventsHandleTime    Timer
}
