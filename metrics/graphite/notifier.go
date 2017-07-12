package graphite

//NotifierMetrics is a collection of metrics used in notifier
type NotifierMetrics struct {
	Config                 Config
	EventsReceived         Meter
	EventsMalformed        Meter
	EventsProcessingFailed Meter
	SubsMalformed          Meter
	SendingFailed          Meter
	SendersOkMetrics       MetricsMap
	SendersFailedMetrics   MetricsMap
}
