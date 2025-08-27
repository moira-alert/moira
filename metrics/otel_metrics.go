package metrics

import (
	"context"
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira"
	"go.opentelemetry.io/otel/attribute"
	internalMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

// DefaultMetricsContext holds metric readers, providers, and attributes.
type DefaultMetricsContext struct {
	readers    []metric.Reader
	providers  []*metric.MeterProvider
	attributes Attributes
}

// NewMetricContext creates a new DefaultMetricsContext.
func NewMetricContext(ctx context.Context) *DefaultMetricsContext {
	return &DefaultMetricsContext{
		readers:    []metric.Reader{},
		providers:  []*metric.MeterProvider{},
		attributes: map[string]string{},
	}
}

// AddReader adds a metric.Reader to the context.
func (d *DefaultMetricsContext) AddReader(reader metric.Reader) {
	d.readers = append(d.readers, reader)
}

// CreateRegistry creates a MetricRegistry from the context's readers.
func (d *DefaultMetricsContext) CreateRegistry() MetricRegistry {
	providers := moira.Map(d.readers, func(reader metric.Reader) *metric.MeterProvider {
		return metric.NewMeterProvider(metric.WithReader(reader))
	})

	return &DefaultMetricRegistry{providers, d.attributes}
}

// Shutdown shuts down all readers and providers in the context.
func (d *DefaultMetricsContext) Shutdown(ctx context.Context) error {
	err := errors.Join(moira.Map(d.readers, func(exp metric.Reader) error { return exp.Shutdown(ctx) })...)
	if err != nil {
		return err
	}

	err = errors.Join(moira.Map(d.providers, func(exp *metric.MeterProvider) error { return exp.Shutdown(ctx) })...)

	return err
}

// DefaultMetricRegistry implements MetricRegistry using MeterProviders and attributes.
type DefaultMetricRegistry struct {
	providers  []*metric.MeterProvider
	attributes Attributes
}

// WithAttributes returns a new MetricRegistry with merged attributes.
func (r *DefaultMetricRegistry) WithAttributes(attributes Attributes) MetricRegistry {
	attrs := make(Attributes, len(r.attributes))
	maps.Copy(attrs, r.attributes)
	maps.Copy(attrs, attributes)

	return &DefaultMetricRegistry{r.providers, attrs}
}

// NewCounter creates a new Counter with the given name.
func (r *DefaultMetricRegistry) NewCounter(name string) Counter {
	counters := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Counter {
		counter, _ := provider.Meter("counter").Int64Counter(name)
		return counter
	})

	return &otelCounter{
		counters,
		0,
		sync.Mutex{},
		r.attributes.toOtelAttributes(),
	}
}

// NewGauge creates a new Gauge with the given name.
func (r *DefaultMetricRegistry) NewGauge(name string) Meter {
	gauges := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Gauge {
		gauge, _ := provider.Meter("gauge").Int64Gauge(name)
		return gauge
	})

	return &otelGauge{
		gauges,
		r.attributes.toOtelAttributes(),
	}
}

// NewHistogram creates a new Histogram with the given name and bucket boundaries.
func (r *DefaultMetricRegistry) NewHistogram(name string) Histogram {
	histograms := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Histogram {
		histogram, _ := provider.Meter("histogram").Int64Histogram(name)
		return histogram
	})

	return &otelHistogram{
		histograms,
		r.attributes.toOtelAttributes(),
	}
}

func (r *DefaultMetricRegistry) NewTimer(name string) Timer {
	timers := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Float64Histogram {
		timer, _ := provider.Meter("timer").Float64Histogram(name)
		return timer
	})

	return &otelTimer{
		timers,
		r.attributes.toOtelAttributes(),
		0,
	}
}

// toOtelAttributes converts Attributes to a slice of attribute.KeyValue.
func (a Attributes) toOtelAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(a))
	for k, v := range a {
		attrs = append(attrs, attribute.String(k, v))
	}

	return attrs
}

// otelCounter implements Counter using OpenTelemetry Int64Counter.
type otelCounter struct {
	counters   []internalMetric.Int64Counter
	count      int64
	mu         sync.Mutex
	attributes []attribute.KeyValue
}

// Inc increments the counter by 1.
func (c *otelCounter) Inc() {
	for _, counter := range c.counters {
		counter.Add(context.Background(), 1, internalMetric.WithAttributes(c.attributes...))
	}

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
	gauges     []internalMetric.Int64Gauge
	attributes []attribute.KeyValue
}

// Mark records a value for the gauge.
func (c *otelGauge) Mark(mark int64) {
	for _, gauge := range c.gauges {
		gauge.Record(context.Background(), mark, internalMetric.WithAttributes(c.attributes...))
	}
}

// otelHistogram implements Histogram using OpenTelemetry Int64Histogram.
type otelHistogram struct {
	histograms []internalMetric.Int64Histogram
	attributes []attribute.KeyValue
}

// Update records a value for the histogram.
func (h *otelHistogram) Update(mark int64) {
	for _, histogram := range h.histograms {
		histogram.Record(context.Background(), mark, internalMetric.WithAttributes(h.attributes...))
	}
}

// otelTimer represents a timer that records durations in histograms with attributes.
type otelTimer struct {
	histogram  []internalMetric.Float64Histogram
	attributes []attribute.KeyValue
	count      int64
}

// UpdateSince records the duration since the given timestamp in all histograms and increments the count.
func (t *otelTimer) UpdateSince(ts time.Time) {
	for _, histogram := range t.histogram {
		histogram.Record(context.Background(), float64(time.Since(ts)), internalMetric.WithAttributes(t.attributes...))
	}

	atomic.AddInt64(&t.count, 1)
}

// Count returns the number of times UpdateSince has been called.
func (t *otelTimer) Count() int64 {
	return atomic.LoadInt64(&t.count)
}
