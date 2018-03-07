package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckError                Meter
	HandleError               Meter
	TriggersCheckTime         Timer
	TriggerCheckTime          TimerMap
	TriggersToCheckChannelLen Meter
	MetricEventsChannelLen    Meter
	MetricEventsHandleTime    Timer
}
