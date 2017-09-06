package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckError       Meter
	HandleError      Meter
	TriggerCheckTime Timer
}
