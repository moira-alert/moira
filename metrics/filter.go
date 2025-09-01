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
func ConfigureFilterMetrics(registry Registry, attributedRegistry MetricRegistry) *FilterMetrics {
	return &FilterMetrics{
		TotalMetricsReceived:        NewCompositeCounter(registry.NewCounter("received", "total"), attributedRegistry.NewCounter("received_total")),
		ValidMetricsReceived:        NewCompositeCounter(registry.NewCounter("received", "valid"), attributedRegistry.NewCounter("received_valid")),
		MatchingMetricsReceived:     NewCompositeCounter(registry.NewCounter("received", "matching"), attributedRegistry.NewCounter("received_matching")),
		PatternMatchingCacheEvicted: NewCompositeMeter(registry.NewMeter("patternMatchingCache", "evicted"), attributedRegistry.NewGauge("patternMatchingCache_evicted")),
		MatchingTimer:               NewCompositeTimer(registry.NewTimer("time", "match"), attributedRegistry.NewTimer("time_match")),
		SavingTimer:                 NewCompositeTimer(registry.NewTimer("time", "save"), attributedRegistry.NewTimer("time_save")),
		BuildTreeTimer:              NewCompositeTimer(registry.NewTimer("time", "buildtree"), attributedRegistry.NewTimer("time_buildtree")),
		MetricChannelLen:            NewCompositeHistogram(registry.NewHistogram("metricsToSave"), attributedRegistry.NewHistogram("metricsToSave")),
		LineChannelLen:              NewCompositeHistogram(registry.NewHistogram("linesToMatch"), attributedRegistry.NewHistogram("linesToMatch")),
	}
}

// MarkPatternMatchingEvicted counts the number of evicted items in the pattern matching cache.
func (metrics *FilterMetrics) MarkPatternMatchingEvicted(evicted int64) {
	metrics.PatternMatchingCacheEvicted.Mark(evicted)
}
