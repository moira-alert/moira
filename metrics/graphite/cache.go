package graphite

// CacheMetrics is a collection of metrics used in cache
type CacheMetrics struct {
	TotalMetricsReceived    Meter // TotalMetricsReceived metrics counter
	ValidMetricsReceived    Meter // ValidMetricsReceived metrics counter
	MatchingMetricsReceived Meter // MatchingMetricsReceived metrics counter
	MatchingTimer           Timer // MatchingTimer metrics timer
	SavingTimer             Timer // SavingTimer metrics timer
	BuildTreeTimer          Timer // BuildTreeTimer metrics timer
}
