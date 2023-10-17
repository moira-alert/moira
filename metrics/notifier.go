package metrics

// NotifierMetrics is a collection of metrics used in notifier
type NotifierMetrics struct {
	SubsMalformed                  Meter
	EventsReceived                 Meter
	EventsMalformed                Meter
	EventsProcessingFailed         Meter
	EventsByState                  MetersCollection
	SendingFailed                  Meter
	SendersOkMetrics               MetersCollection
	SendersFailedMetrics           MetersCollection
	PlotsBuildDurationMs           Histogram
	PlotsEvaluateTriggerDurationMs Histogram
	FetchNotificationsDurationMs   Histogram
}

// ConfigureNotifierMetrics is notifier metrics configurator
func ConfigureNotifierMetrics(registry Registry, prefix string) *NotifierMetrics {
	return &NotifierMetrics{
		SubsMalformed:                  registry.NewMeter("subs", "malformed"),
		EventsReceived:                 registry.NewMeter("events", "received"),
		EventsMalformed:                registry.NewMeter("events", "malformed"),
		EventsProcessingFailed:         registry.NewMeter("events", "failed"),
		EventsByState:                  NewMetersCollection(registry),
		SendingFailed:                  registry.NewMeter("sending", "failed"),
		SendersOkMetrics:               NewMetersCollection(registry),
		SendersFailedMetrics:           NewMetersCollection(registry),
		PlotsBuildDurationMs:           registry.NewHistogram("plots", "build", "duration", "ms"),
		PlotsEvaluateTriggerDurationMs: registry.NewHistogram("plots", "evaluate", "trigger", "duration", "ms"),
		FetchNotificationsDurationMs:   registry.NewHistogram("fetch", "notifications", "duration", "ms"),
	}
}
