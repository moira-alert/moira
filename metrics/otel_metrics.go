package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	internalMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type defaultMetricsContextModifier interface {
	modify(*DefaultMetricsContext) error
}

// DefaultMetricsContext holds metric readers, providers, and attributes.
type DefaultMetricsContext struct {
	readers   []metric.Reader
	provider  *metric.MeterProvider
	modifiers []defaultMetricsContextModifier
}

// NewMetricContext creates a new DefaultMetricsContext.
func NewMetricContext(ctx context.Context) *DefaultMetricsContext {
	return &DefaultMetricsContext{
		readers:   []metric.Reader{},
		provider:  &metric.MeterProvider{},
		modifiers: []defaultMetricsContextModifier{},
	}
}

// AddReader adds a metric.Reader to the context.
func (d *DefaultMetricsContext) AddReader(reader metric.Reader) {
	d.readers = append(d.readers, reader)
}

type runtimeStatsModifier struct {
	samplingInterval time.Duration
}

func (m *runtimeStatsModifier) modify(c *DefaultMetricsContext) error {
	err := runtime.Start(
		runtime.WithMeterProvider(c.provider),
		runtime.WithMinimumReadMemStatsInterval(m.samplingInterval),
	)

	return err
}

// AddRuntimeStats appends runtime statistics collection to the metrics context.
func (d *DefaultMetricsContext) AddRuntimeStats(samplingRate time.Duration) {
	d.modifiers = append(d.modifiers, &runtimeStatsModifier{samplingRate})
}

// CreateRegistry creates a MetricRegistry from the context's readers.
func (d *DefaultMetricsContext) CreateRegistry(attributes ...Attribute) (MetricRegistry, error) {
	opts := make([]metric.Option, 0, len(d.readers))
	for _, r := range d.readers {
		opts = append(opts, metric.WithReader(r))
	}

	opts = append(opts, metric.WithResource(
		resource.NewWithAttributes(semconv.SchemaURL, Attributes(attributes).toOtelAttributes()...),
	))
	provider := metric.NewMeterProvider(opts...)
	d.provider = provider

	for _, modifier := range d.modifiers {
		err := modifier.modify(d)
		if err != nil {
			return nil, err
		}
	}

	return &DefaultMetricRegistry{provider, Attributes{}}, nil
}

// Shutdown shuts down all readers and providers in the context.
func (d *DefaultMetricsContext) Shutdown(ctx context.Context) error {
	err := d.provider.Shutdown(ctx)
	return err
}

// DefaultMetricRegistry implements MetricRegistry using MeterProviders and attributes.
type DefaultMetricRegistry struct {
	provider   *metric.MeterProvider
	attributes Attributes
}

// WithAttributes returns a new MetricRegistry with merged attributes.
func (r *DefaultMetricRegistry) WithAttributes(attributes Attributes) MetricRegistry {
	attrs := make(Attributes, 0, len(r.attributes)+len(attributes))
	attrs = append(attrs, r.attributes...)
	attrs = append(attrs, attributes...)

	return &DefaultMetricRegistry{r.provider, attrs}
}

// NewCounter creates a new Counter with the given name.
func (r *DefaultMetricRegistry) NewCounter(name string) (Counter, error) {
	counter, err := r.provider.Meter("counter").Int64Counter(name)
	if err != nil {
		return nil, err
	}

	return &otelCounter{
		counter,
		0,
		sync.Mutex{},
		r.attributes.toOtelAttributes(),
	}, nil
}

// NewGauge creates a new Gauge with the given name.
func (r *DefaultMetricRegistry) NewGauge(name string) (Meter, error) {
	gauge, err := r.provider.Meter("gauge").Int64Gauge(name)
	if err != nil {
		return nil, err
	}

	return &otelGauge{
		gauge,
		r.attributes.toOtelAttributes(),
	}, nil
}

// NewHistogram creates a new Histogram with the given name.
func (r *DefaultMetricRegistry) NewHistogram(name string) (Histogram, error) {
	histogram, err := r.provider.Meter("histogram").Int64Histogram(name)
	if err != nil {
		return nil, err
	}

	return &otelHistogram{
		histogram,
		r.attributes.toOtelAttributes(),
	}, nil
}

// NewTimer creates a new Timer with the given name.
func (r *DefaultMetricRegistry) NewTimer(name string) (Timer, error) {
	timer, err := r.provider.Meter("timer").Float64Histogram(
		name,
		internalMetric.WithExplicitBucketBoundaries(0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10, 20, 100, 1000),
	)
	if err != nil {
		return nil, err
	}

	return &otelTimer{
		timer,
		r.attributes.toOtelAttributes(),
		0,
	}, nil
}

// toOtelAttributes converts Attributes to a slice of attribute.KeyValue.
func (attributes Attributes) toOtelAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for _, attr := range attributes {
		attrs = append(attrs, attribute.String(attr.Key, attr.Value))
	}

	return attrs
}

// otelCounter implements Counter using OpenTelemetry Int64Counter.
type otelCounter struct {
	counter    internalMetric.Int64Counter
	count      int64
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// Inc increments the counter by 1.
func (c *otelCounter) Inc() {
	c.counter.Add(context.Background(), 1, internalMetric.WithAttributes(c.attributes...))

	c.mu.Lock()
	defer c.mu.Unlock()

	c.count++
}

// Count returns the current count value.
func (c *otelCounter) Count() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.count
}

// otelGauge implements Meter using OpenTelemetry Int64Gauge.
type otelGauge struct {
	gauge      internalMetric.Int64Gauge
	attributes []attribute.KeyValue
}

// Mark records a value for the gauge.
func (c *otelGauge) Mark(mark int64) {
	c.gauge.Record(context.Background(), mark, internalMetric.WithAttributes(c.attributes...))
}

// otelHistogram implements Histogram using OpenTelemetry Int64Histogram.
type otelHistogram struct {
	histogram  internalMetric.Int64Histogram
	attributes []attribute.KeyValue
}

// Update records a value for the histogram.
func (h *otelHistogram) Update(mark int64) {
	h.histogram.Record(context.Background(), mark, internalMetric.WithAttributes(h.attributes...))
}

// otelTimer represents a timer that records durations in histograms with attributes.
type otelTimer struct {
	histogram  internalMetric.Float64Histogram
	attributes []attribute.KeyValue
	count      int64
}

// UpdateSince records the duration since the given timestamp in all histograms and increments the count.
func (t *otelTimer) UpdateSince(ts time.Time) {
	t.histogram.Record(context.Background(), time.Since(ts).Seconds(), internalMetric.WithAttributes(t.attributes...))

	atomic.AddInt64(&t.count, 1)
}

// Count returns the number of times UpdateSince has been called.
func (t *otelTimer) Count() int64 {
	return atomic.LoadInt64(&t.count)
}

// DefaultAttributedMetricCollection represents a collection of attributed metrics with default behavior.
type DefaultAttributedMetricCollection struct {
	registry MetricRegistry
	meters   map[string]Meter
	counters map[string]Counter
}

// NewAttributedMetricCollection creates a new AttributedMetricCollection with the given registry.
func NewAttributedMetricCollection(registry MetricRegistry) AttributedMetricCollection {
	return &DefaultAttributedMetricCollection{
		registry: registry,
		meters:   map[string]Meter{},
		counters: map[string]Counter{},
	}
}

// RegisterMeter registers a new meter with the specified name, metric, and attributes.
func (r *DefaultAttributedMetricCollection) RegisterMeter(name string, metric string, attributes Attributes) (Meter, error) {
	gauge, err := r.registry.WithAttributes(attributes).NewGauge(metric)
	if err != nil {
		return nil, err
	}

	r.meters[name] = gauge

	return gauge, nil
}

// RegisterCounter registers a new counter with the specified name, metric, and attributes.
func (r *DefaultAttributedMetricCollection) RegisterCounter(name string, metric string, attributes Attributes) (Counter, error) {
	counter, err := r.registry.WithAttributes(attributes).NewCounter(metric)
	if err != nil {
		return nil, err
	}

	r.counters[name] = counter

	return counter, nil
}

// GetRegisteredMeter retrieves a registered meter by name.
func (r *DefaultAttributedMetricCollection) GetRegisteredMeter(name string) (Meter, bool) {
	gauge, ok := r.meters[name]
	return gauge, ok
}

// GetRegisteredCounter retrieves a registered counter by name.
func (r *DefaultAttributedMetricCollection) GetRegisteredCounter(name string) (Counter, bool) {
	gauge, ok := r.counters[name]
	return gauge, ok
}
