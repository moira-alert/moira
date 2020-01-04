package metrics

// NotifierMetrics is a collection of metrics used in notifier
type NotifierMetrics struct {
	SubsMalformed          Meter
	EventsReceived         Meter
	EventsMalformed        Meter
	EventsProcessingFailed Meter
	SendingFailed          Meter
	SendersOkMetrics       MetersCollection
	SendersFailedMetrics   MetersCollection
}

// ConfigureNotifierMetrics is notifier metrics configurator
func ConfigureNotifierMetrics(registry Registry, prefix string) *NotifierMetrics {
	return &NotifierMetrics{
		SubsMalformed:          registry.NewMeter("subs", "malformed"),
		EventsReceived:         registry.NewMeter("events", "received"),
		EventsMalformed:        registry.NewMeter("events", "malformed"),
		EventsProcessingFailed: registry.NewMeter("events", "failed"),
		SendingFailed:          registry.NewMeter("sending", "failed"),
		SendersOkMetrics:       NewMetersCollection(registry),
		SendersFailedMetrics:   NewMetersCollection(registry),
	}
}
