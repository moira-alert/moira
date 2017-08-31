package graphite

// CheckerMetrics is a collection of metrics used in checker
type CheckerMetrics struct {
	CheckerError      Meter
	TriggerCheckTime  Timer
	TriggerCheckGauge Gauge
}
