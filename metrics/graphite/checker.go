package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	MoiraMetrics           *CheckMetrics
	RemoteMetrics          *CheckMetrics
	MetricEventsChannelLen Histogram
	MetricEventsHandleTime Timer
}

type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggerCheckTime     TimerMap
	TriggersToCheckCount Histogram
}
