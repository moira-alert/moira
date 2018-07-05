package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckError                      Meter
	HandleError                     Meter
	RemoteHandleError               Meter
	TriggersCheckTime               Timer
	TriggerCheckTime                TimerMap
	RemoteTriggersCheckTime         Timer
	RemoteTriggerCheckTime          TimerMap
	TriggersToCheckChannelLen       Histogram
	RemoteTriggersToCheckChannelLen Histogram
	MetricEventsChannelLen          Histogram
	MetricEventsHandleTime          Timer
}
