package metrics

import (
	"context"
	"errors"

	"github.com/moira-alert/moira"
	"go.opentelemetry.io/otel/attribute"
	internalMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"

	"maps"
)

type DefaultMetricsContext struct {
	readers []metric.Reader
	providers []*metric.MeterProvider
	attributes Attributes
}

func NewMetricContext(ctx context.Context) *DefaultMetricsContext {
	return &DefaultMetricsContext{
		readers: []metric.Reader{},
		providers: []*metric.MeterProvider{},
		attributes: map[string]string{},

	}
}

func (d *DefaultMetricsContext) AddReader(reader metric.Reader) {
	d.readers = append(d.readers, reader)
}

func (d *DefaultMetricsContext) CreateRegistry() MetricRegistry {
	providers := moira.Map(d.readers, func(reader metric.Reader) *metric.MeterProvider {
		return metric.NewMeterProvider(metric.WithReader(reader))
	})
	return &DefaultMetricRegistry{providers, d.attributes}
}


func (d *DefaultMetricsContext) Shutdown(ctx context.Context) error {
	err := errors.Join(moira.Map(d.readers, func(exp metric.Reader) error { return exp.Shutdown(ctx) })...)
	if err != nil {
		return err
	}
	err = errors.Join(moira.Map(d.providers, func(exp *metric.MeterProvider) error { return exp.Shutdown(ctx) })...)
	return err
}


type DefaultMetricRegistry struct {
	providers []*metric.MeterProvider
	attributes Attributes
}

func (r *DefaultMetricRegistry) WithAttributes(attributes Attributes) MetricRegistry {
	attrs := make(Attributes, len(r.attributes))
	maps.Copy(attrs, r.attributes)
	maps.Copy(attrs, attributes)
	return &DefaultMetricRegistry{r.providers, attrs}
}

func (r *DefaultMetricRegistry) NewCounter(name string) Counter {
	counters := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Counter {
		counter, _ := provider.Meter("counter").Int64Counter(name)
		return counter
	})

	return &OtelCounter{
		counters,
		0,
		r.attributes.ToOtelAttributes(),
	}
}

func (r *DefaultMetricRegistry) NewGauge(name string) Meter {
	gauges := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Gauge {
		gauge, _ := provider.Meter("gauge").Int64Gauge(name)
		return gauge
	})

	return &OtelGauge{
		gauges,
		r.attributes.ToOtelAttributes(),
	}
}

func (r *DefaultMetricRegistry) NewHistogram(bucketBoundaries []uint64, name string) Histogram {
	histograms := moira.Map(r.providers, func(provider *metric.MeterProvider) internalMetric.Int64Histogram {
		histogram, _ := provider.Meter("histogram").Int64Histogram(name)
		return histogram
	})
	return &OtelHistogram{
		histograms,
		r.attributes.ToOtelAttributes(),
	}
}

func (a *Attributes) ToOtelAttributes() []attribute.KeyValue {
	if a == nil {
		return nil
	}
	attrs := make([]attribute.KeyValue, 0, len(*a))
	for k, v := range *a {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}

// Counter

type OtelCounter struct {
	counters []internalMetric.Int64Counter
	count int64
	attributes []attribute.KeyValue
}

func (c *OtelCounter) Inc() {
	for _, counter := range c.counters {
		counter.Add(context.Background(), 1, internalMetric.WithAttributes(c.attributes...))
	}
	c.count += 1
}

func (c *OtelCounter) Count() int64 {
	return c.count
}

// Gauge

type OtelGauge struct {
	gauges []internalMetric.Int64Gauge
	attributes []attribute.KeyValue
}

func (c *OtelGauge) Mark(mark int64) {
	for _, gauge := range c.gauges {
		gauge.Record(context.Background(), mark, internalMetric.WithAttributes(c.attributes...))
	}
}

// Histogram

type OtelHistogram struct {
	histograms []internalMetric.Int64Histogram
	attributes []attribute.KeyValue
}

func (h *OtelHistogram) Update(mark int64) {
	for _, histogram := range h.histograms {
		histogram.Record(context.Background(), mark, internalMetric.WithAttributes(h.attributes...))
	}
}

