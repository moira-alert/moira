package graphite

// FilterMetrics is a collection of metrics used in filter
type FilterMetrics struct {
	TotalMetricsReceived    Counter
	ValidMetricsReceived    Counter
	MatchingMetricsReceived Counter
	MatchingTimer           Timer
	SavingTimer             Timer
	BuildTreeTimer          Timer
	MetricChannelLen        Histogram
}
