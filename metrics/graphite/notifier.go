package graphite

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// NotifierMetrics is a collection of metrics used in notifier
type NotifierMetrics struct {
	RuntimeMetricsRegistry *goMetrics.Registry
	SubsMalformed          Meter
	EventsReceived         Meter
	EventsMalformed        Meter
	EventsProcessingFailed Meter
	SendingFailed          Meter
	SendersOkMetrics       MetricsMap
	SendersFailedMetrics   MetricsMap
}
