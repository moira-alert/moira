package metrics

const prefix = "searchIndex"

// IndexMetrics is a collection of metrics used in full-text search index.
type IndexMetrics struct {
	IndexedTriggersCount  Histogram
	IndexActualizationLag Timer
}

// ConfigureIndexMetrics in full-text search index metrics configurator.
func ConfigureIndexMetrics(registry Registry, attributedRegistry MetricRegistry, settings Settings) (*IndexMetrics, error) {
	const indexedTriggersCountMetric string = "index.triggers.count"

	indexedTriggersCount, err := attributedRegistry.NewHistogram(indexedTriggersCountMetric, settings.GetHistogramBucketOr(indexedTriggersCountMetric, DefaultHistogramBackets))
	if err != nil {
		return nil, err
	}

	const actualizationLagMetric string = "index.actualization_lag"

	actualizationLag, err := attributedRegistry.NewTimer(actualizationLagMetric, settings.GetTimerBucketOr(actualizationLagMetric, DefaultTimerBackets))
	if err != nil {
		return nil, err
	}

	return &IndexMetrics{
		IndexedTriggersCount:  NewCompositeHistogram(registry.NewHistogram(prefix, "indexedTriggers"), indexedTriggersCount),
		IndexActualizationLag: NewCompositeTimer(registry.NewTimer(prefix, "actualizationLag"), actualizationLag),
	}, nil
}
