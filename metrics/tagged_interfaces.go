package metrics

import "context"

// Attribute represents a key-value string pair for metric attributes.
type Attribute struct {
	// key is the attribute's key
	Key string
	// value is the attribute's value
	Value string
}

// Attributes represents a set of key-value string pairs for metric attributes.
type Attributes []Attribute

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
	NewCounter(name string) (Counter, error)
	// NewGauge creates and returns a new Meter gauge metric with the given name.
	NewGauge(name string) (Meter, error)
	// NewHistogram creates and returns a new Histogram metric with the given name.
	NewHistogram(name string) (Histogram, error)
	// NewTimer creates and returns a new Timer metric with the given name.
	NewTimer(name string) (Timer, error)
}

// AttributedMetricCollection represents a collection of attributed meters.
type AttributedMetricCollection interface {
	// RegisterMeter registers a new Meter with the given name and attributes.
	RegisterMeter(name string, attributes Attributes) (Meter, error)
	// GetRegisteredMeter retrieves a registered Meter by name.
	GetRegisteredMeter(name string) (Meter, bool)
}
