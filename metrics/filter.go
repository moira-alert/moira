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
func ConfigureFilterMetrics(registry Registry, attributedRegistry MetricRegistry) (*FilterMetrics, error) {
	totalMetricsReceived, err := attributedRegistry.NewCounter("received.total")
	if err != nil {
		return nil, err
	}

	validMetricsReceived, err := attributedRegistry.NewCounter("received.valid")
	if err != nil {
		return nil, err
	}

	matchingMetricsReceived, err := attributedRegistry.NewCounter("received.matching")
	if err != nil {
		return nil, err
	}

	patternMatchingCacheEvicted, err := attributedRegistry.NewGauge("pattern_matching_cache.evicted")
	if err != nil {
		return nil, err
	}

	matchingTimer, err := attributedRegistry.NewTimer("time.match")
	if err != nil {
		return nil, err
	}

	savingTimer, err := attributedRegistry.NewTimer("time.save")
	if err != nil {
		return nil, err
	}

	buildTreeTimer, err := attributedRegistry.NewTimer("time.buildtree")
	if err != nil {
		return nil, err
	}

	metricChannelLen, err := attributedRegistry.NewHistogram("channel.metric.to_save.len")
	if err != nil {
		return nil, err
	}

	linesToMatch, err := attributedRegistry.NewHistogram("channel.lines.to_match.len")
	if err != nil {
		return nil, err
	}

	return &FilterMetrics{
		// Deprecated: only received.total metric of attributedRegistry should be used.
		TotalMetricsReceived: NewCompositeCounter(registry.NewCounter("received", "total"), totalMetricsReceived),
		// Deprecated: only received.valid metric of attributedRegistry should be used.
		ValidMetricsReceived: NewCompositeCounter(registry.NewCounter("received", "valid"), validMetricsReceived),
		// Deprecated: only received.matching metric of attributedRegistry should be used.
		MatchingMetricsReceived: NewCompositeCounter(registry.NewCounter("received", "matching"), matchingMetricsReceived),
		// Deprecated: only pattern_matching_cache.evicted metric of attributedRegistry should be used.
		PatternMatchingCacheEvicted: NewCompositeMeter(registry.NewMeter("patternMatchingCache", "evicted"), patternMatchingCacheEvicted),
		// Deprecated: only time.match metric of attributedRegistry should be used.
		MatchingTimer: NewCompositeTimer(registry.NewTimer("time", "match"), matchingTimer),
		// Deprecated: only time.save metric of attributedRegistry should be used.
		SavingTimer: NewCompositeTimer(registry.NewTimer("time", "save"), savingTimer),
		// Deprecated: only time.buildtree metric of attributedRegistry should be used.
		BuildTreeTimer: NewCompositeTimer(registry.NewTimer("time", "buildtree"), buildTreeTimer),
		// Deprecated: only channel.metric.to_save.len metric of attributedRegistry should be used.
		MetricChannelLen: NewCompositeHistogram(registry.NewHistogram("metricsToSave"), metricChannelLen),
		// Deprecated: only channel.lines.to_match.len metric of attributedRegistry should be used.
		LineChannelLen: NewCompositeHistogram(registry.NewHistogram("linesToMatch"), linesToMatch),
	}, nil
}

// MarkPatternMatchingEvicted counts the number of evicted items in the pattern matching cache.
func (metrics *FilterMetrics) MarkPatternMatchingEvicted(evicted int64) {
	metrics.PatternMatchingCacheEvicted.Mark(evicted)
}
