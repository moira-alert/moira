package metrics

const prefix = "searchIndex"

// IndexMetrics is a collection of metrics used in full-text search index.
type IndexMetrics struct {
	IndexedTriggersCount  Histogram
	IndexActualizationLag Timer
}

// ConfigureIndexMetrics in full-text search index metrics configurator.
func ConfigureIndexMetrics(registry Registry, attributedRegistry MetricRegistry) (*IndexMetrics, error) {
	indexedTriggersCount, err := attributedRegistry.NewHistogram("index.triggers.count")
	if err != nil {
		return nil, err
	}

	actualizationLag, err := attributedRegistry.NewTimer("index.actualization_lag")
	if err != nil {
		return nil, err
	}

	return &IndexMetrics{
		IndexedTriggersCount:  NewCompositeHistogram(registry.NewHistogram(prefix, "indexedTriggers"), indexedTriggersCount),
		IndexActualizationLag: NewCompositeTimer(registry.NewTimer(prefix, "actualizationLag"), actualizationLag),
	}, nil
}
