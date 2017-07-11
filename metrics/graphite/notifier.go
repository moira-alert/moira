package graphite

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
