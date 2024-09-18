package metrics

const prefix = "searchIndex"

// IndexMetrics is a collection of metrics used in full-text search index.
type IndexMetrics struct {
	IndexedTriggersCount  Histogram
	IndexActualizationLag Timer
}

// ConfigureIndexMetrics in full-text search index metrics configurator.
func ConfigureIndexMetrics(registry Registry) *IndexMetrics {
	return &IndexMetrics{
		IndexedTriggersCount:  registry.NewHistogram(prefix, "indexedTriggers"),
		IndexActualizationLag: registry.NewTimer(prefix, "actualizationLag"),
	}
}
