package metrics

import "context"

// Attributes represents a set of key-value string pairs for metric attributes.
type Attributes map[string]string

// MetricsContext provides methods to create a metric registry and shutdown the context.
type MetricsContext interface {
	// CreateRegistry creates and returns a new MetricRegistry.
	CreateRegistry() MetricRegistry
	// Shutdown gracefully shuts down the metrics context.
	Shutdown(ctx context.Context) error
}

// MetricRegistry provides methods to create and manage metrics with attributes.
type MetricRegistry interface {
	// WithAttributes returns a new MetricRegistry with the given attributes.
	WithAttributes(attributes Attributes) MetricRegistry
	// NewCounter creates and returns a new Counter metric with the given name.
	NewCounter(name string) Counter
	// NewGauge creates and returns a new Meter gauge metric with the given name.
	NewGauge(name string) Meter
	// NewHistogram creates and returns a new Histogram metric with the given bucket boundaries and name.
	NewHistogram(bucketBoundaries []uint64, name string) Histogram
}
