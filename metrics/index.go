package metrics

// IndexMetrics is a collection of metrics used in full-text search index
type IndexMetrics struct {
	IndexedTriggersCount  Histogram
	IndexActualizationLag Timer
}

// ConfigureIndexMetrics in full-text search index metrics configurator
func ConfigureIndexMetrics(registry Registry, prefix string) *IndexMetrics {
	return &IndexMetrics{
		IndexedTriggersCount:  registry.NewHistogram(metricNameWithPrefix(prefix, "indexedTriggers")),
		IndexActualizationLag: registry.NewTimer(metricNameWithPrefix(prefix, "actualizationLag")),
	}
}
