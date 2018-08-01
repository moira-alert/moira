package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckError        Meter
	HandleError       Meter
	TriggersCheckTime Timer
	TriggerCheckTime  TimerMap
	// ToDo: add TriggersToCheckChannelLen
	MetricEventsChannelLen Histogram
	MetricEventsHandleTime Timer

	RemoteHandleError               Meter
	RemoteTriggersCheckTime         Timer
	RemoteTriggerCheckTime          TimerMap
	RemoteTriggersToCheckChannelLen Histogram
}
