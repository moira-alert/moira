package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckError             Meter
	HandleError            Meter
	TriggersCheckTime      Timer
	TriggerCheckTime       TimerMap
	TriggersToCheckCount   Histogram
	MetricEventsChannelLen Histogram
	MetricEventsHandleTime Timer

	RemoteHandleError          Meter
	RemoteTriggersCheckTime    Timer
	RemoteTriggerCheckTime     TimerMap
	RemoteTriggersToCheckCount Histogram
}
