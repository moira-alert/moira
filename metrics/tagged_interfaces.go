package metrics

import "context"

type Attributes map[string]string

type MetricsContext interface {
	CreateRegistry() MetricRegistry
	Shutdown(ctx context.Context) error
}

type MetricRegistry interface {
	WithAttributes(attributes Attributes) MetricRegistry
	NewCounter(name string) Counter
	NewGauge(name string) Meter
	NewHistogram(bucketBoundaries []uint64, name string) Histogram
}

