package graphite

// IndexMetrics is a collection of metrics used in full-text search index
type IndexMetrics struct {
	IndexedTriggersCount  Histogram
	IndexActualizationLag Timer
}
