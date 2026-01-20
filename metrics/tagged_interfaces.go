package metrics

import "context"

var (
	DefaultHistogramBackets []int64   = []int64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000}
	DefaultTimerBackets     []float64 = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 1.25, 2, 2.5, 5, 7.5, 10, 20, 100, 1000}
)

// Attribute represents a key-value string pair for metric attributes.
type Attribute struct {
	// key is the attribute's key
	Key string
	// value is the attribute's value
	Value string
}

// Attributes represents a set of key-value string pairs for metric attributes.
type (
	Attributes     []Attribute
	Buckets[T any] map[string][]T
	Settings       struct {
		HistogramBuckets Buckets[int64]
		TimerBuckets     Buckets[float64]
	}
)

// MetricsContext provides methods to create a metric registry and shutdown the context.
type MetricsContext interface {
	// CreateRegistry creates and returns a new MetricRegistry.
	CreateRegistry(attributes ...Attribute) (MetricRegistry, error)
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
	// NewHistogram creates and returns a new Histogram metric with the given name and buckets.
	NewHistogram(name string, buckets []int64) (Histogram, error)
	// NewTimer creates and returns a new Timer metric with the given name and buckets.
	NewTimer(name string, buckets []float64) (Timer, error)
}

// AttributedMetricCollection represents a collection of attributed meters.
type AttributedMetricCollection interface {
	// RegisterMeter registers a new Meter with the given name, metric and attributes.
	RegisterMeter(name string, metric string, attributes Attributes) (Meter, error)
	// GetRegisteredMeter retrieves a registered Meter by name.
	GetRegisteredMeter(name string) (Meter, bool)
	// RegisterCounter registers a new Counter with the given name, metric and attributes.
	RegisterCounter(name string, metric string, attributes Attributes) (Counter, error)
	// GetRegisteredCounter retrieves a registered Counter by name.
	GetRegisteredCounter(name string) (Counter, bool)
}
