package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	MoiraMetrics           *CheckMetrics
	RemoteMetrics          *CheckMetrics
	MetricEventsChannelLen Histogram
	MetricEventsHandleTime Timer
}

// CheckMetrics is a collection of metrics for trigger checks
type CheckMetrics struct {
	CheckError           Meter
	HandleError          Meter
	TriggersCheckTime    Timer
	TriggerCheckTime     TimerMap
	TriggersToCheckCount Histogram
}
