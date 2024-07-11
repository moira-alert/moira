package metrics

// FilterMetrics is a collection of metrics used in filter.
type FilterMetrics struct {
	TotalMetricsReceived        Counter
	ValidMetricsReceived        Counter
	MatchingMetricsReceived     Counter
	PatternMatchingCacheEvicted Meter
	MatchingTimer               Timer
	SavingTimer                 Timer
	BuildTreeTimer              Timer
	MetricChannelLen            Histogram
	LineChannelLen              Histogram
}

// ConfigureFilterMetrics initialize metrics.
func ConfigureFilterMetrics(registry Registry) *FilterMetrics {
	return &FilterMetrics{
		TotalMetricsReceived:        registry.NewCounter("received", "total"),
		ValidMetricsReceived:        registry.NewCounter("received", "valid"),
		MatchingMetricsReceived:     registry.NewCounter("received", "matching"),
		PatternMatchingCacheEvicted: registry.NewMeter("patternMatchingCache", "evicted"),
		MatchingTimer:               registry.NewTimer("time", "match"),
		SavingTimer:                 registry.NewTimer("time", "save"),
		BuildTreeTimer:              registry.NewTimer("time", "buildtree"),
		MetricChannelLen:            registry.NewHistogram("metricsToSave"),
		LineChannelLen:              registry.NewHistogram("linesToMatch"),
	}
}

// MarkPatternMatchingEvicted counts the number of evicted items in the pattern matching cache.
func (metrics *FilterMetrics) MarkPatternMatchingEvicted(evicted int64) {
	metrics.PatternMatchingCacheEvicted.Mark(evicted)
}
