package metrics

// FilterMetrics is a collection of metrics used in filter
type FilterMetrics struct {
	TotalMetricsReceived    Counter
	ValidMetricsReceived    Counter
	MatchingMetricsReceived Counter
	MatchingTimer           Timer
	SavingTimer             Timer
	BuildTreeTimer          Timer
	MetricChannelLen        Histogram
	LineChannelLen          Histogram
}

// ConfigureFilterMetrics initialize metrics
func ConfigureFilterMetrics(registry Registry, prefix string) *FilterMetrics {
	return &FilterMetrics{
		TotalMetricsReceived:    registry.NewCounter(metricNameWithPrefix(prefix, "received.total")),
		ValidMetricsReceived:    registry.NewCounter(metricNameWithPrefix(prefix, "received.valid")),
		MatchingMetricsReceived: registry.NewCounter(metricNameWithPrefix(prefix, "received.matching")),
		MatchingTimer:           registry.NewTimer(metricNameWithPrefix(prefix, "time.match")),
		SavingTimer:             registry.NewTimer(metricNameWithPrefix(prefix, "time.save")),
		BuildTreeTimer:          registry.NewTimer(metricNameWithPrefix(prefix, "time.buildtree")),
		MetricChannelLen:        registry.NewHistogram(metricNameWithPrefix(prefix, "metricsToSave")),
		LineChannelLen:          registry.NewHistogram(metricNameWithPrefix(prefix, "linesToMatch")),
	}
}
