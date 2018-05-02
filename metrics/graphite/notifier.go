package graphite

// NotifierMetrics is a collection of metrics used in notifier
type NotifierMetrics struct {
	SubsMalformed          Meter
	EventsReceived         Meter
	EventsMalformed        Meter
	EventsProcessingFailed Meter
	SendingFailed          Meter
	SendersOkMetrics       MetricsMap
	SendersFailedMetrics   MetricsMap
}
