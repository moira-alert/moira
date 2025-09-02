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
	totalMetricsReceived, err := attributedRegistry.NewCounter("received_total")
	if err != nil {
		return nil, err
	}

	validMetricsReceived, err := attributedRegistry.NewCounter("received_valid")
	if err != nil {
		return nil, err
	}

	matchingMetricsReceived, err := attributedRegistry.NewCounter("received_matching")
	if err != nil {
		return nil, err
	}

	patternMatchingCacheEvicted, err := attributedRegistry.NewGauge("patternMatchingCache_evicted")
	if err != nil {
		return nil, err
	}

	matchingTimer, err := attributedRegistry.NewTimer("time_match")
	if err != nil {
		return nil, err
	}

	savingTimer, err := attributedRegistry.NewTimer("time_save")
	if err != nil {
		return nil, err
	}

	buildTreeTimer, err := attributedRegistry.NewTimer("time_buildtree")
	if err != nil {
		return nil, err
	}

	metricChannelLen, err := attributedRegistry.NewHistogram("metricsToSave")
	if err != nil {
		return nil, err
	}

	lineChannelLen, err := attributedRegistry.NewHistogram("linesToMatch")
	if err != nil {
		return nil, err
	}

	return &FilterMetrics{
		TotalMetricsReceived:        NewCompositeCounter(registry.NewCounter("received", "total"), totalMetricsReceived),
		ValidMetricsReceived:        NewCompositeCounter(registry.NewCounter("received", "valid"), validMetricsReceived),
		MatchingMetricsReceived:     NewCompositeCounter(registry.NewCounter("received", "matching"), matchingMetricsReceived),
		PatternMatchingCacheEvicted: NewCompositeMeter(registry.NewMeter("patternMatchingCache", "evicted"), patternMatchingCacheEvicted),
		MatchingTimer:               NewCompositeTimer(registry.NewTimer("time", "match"), matchingTimer),
		SavingTimer:                 NewCompositeTimer(registry.NewTimer("time", "save"), savingTimer),
		BuildTreeTimer:              NewCompositeTimer(registry.NewTimer("time", "buildtree"), buildTreeTimer),
		MetricChannelLen:            NewCompositeHistogram(registry.NewHistogram("metricsToSave"), metricChannelLen),
		LineChannelLen:              NewCompositeHistogram(registry.NewHistogram("linesToMatch"), lineChannelLen),
	}, nil
}

// MarkPatternMatchingEvicted counts the number of evicted items in the pattern matching cache.
func (metrics *FilterMetrics) MarkPatternMatchingEvicted(evicted int64) {
	metrics.PatternMatchingCacheEvicted.Mark(evicted)
}
