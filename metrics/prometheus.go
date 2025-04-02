package metrics

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const namespace = "moira"

func NewPrometheusRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return registry
}

type PrometheusRegistryAdapter struct {
	registry *prometheus.Registry
	service  string
}

func NewPrometheusRegistryAdapter(registry *prometheus.Registry, service string) *PrometheusRegistryAdapter {
	return &PrometheusRegistryAdapter{registry, service}
}

func (source *PrometheusRegistryAdapter) NewTimer(path ...string) Timer {
	histogramOpts := prometheus.HistogramOpts{Namespace: namespace, Subsystem: source.service, Name: getPrometheusMetricName(path)}
	histogram := prometheus.NewHistogram(histogramOpts)
	source.registry.MustRegister(histogram)
	return &prometheusTimer{histogram: histogram}
}

func (source *PrometheusRegistryAdapter) NewMeter(path ...string) Meter {
	summaryOpts := prometheus.SummaryOpts{Namespace: namespace, Subsystem: source.service, Name: getPrometheusMetricName(path)}
	summary := prometheus.NewSummary(summaryOpts)
	source.registry.MustRegister(summary)
	return &prometheusMeter{summary: summary}
}

func (source *PrometheusRegistryAdapter) NewCounter(path ...string) Counter {
	counterOpts := prometheus.CounterOpts{Namespace: namespace, Subsystem: source.service, Name: getPrometheusMetricName(path)}
	counter := prometheus.NewCounter(counterOpts)
	source.registry.MustRegister(counter)
	return &prometheusCounter{counter: counter}
}

func (source *PrometheusRegistryAdapter) NewHistogram(path ...string) Histogram {
	histogramOpts := prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: source.service,
		Name:      getPrometheusMetricName(path),
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10, 20, 100, 200, 300, 500, 1000},
	}
	histogram := prometheus.NewHistogram(histogramOpts)
	source.registry.MustRegister(histogram)
	return &prometheusHistogram{histogram: histogram}
}

type prometheusHistogram struct {
	count     int64
	histogram prometheus.Histogram
}

func (source *prometheusHistogram) Update(value int64) {
	atomic.AddInt64(&source.count, 1)
	source.histogram.Observe(float64(value))
}

func (source *prometheusHistogram) Count() int64 {
	return atomic.LoadInt64(&source.count)
}

type prometheusMeter struct {
	count   int64
	summary prometheus.Summary
}

func (source *prometheusMeter) Mark(value int64) {
	atomic.AddInt64(&source.count, 1)
	source.summary.Observe(float64(value))
}

func (source *prometheusMeter) Count() int64 {
	return atomic.LoadInt64(&source.count)
}

type prometheusTimer struct {
	histogram prometheus.Histogram
	count     int64
}

func (source *prometheusTimer) UpdateSince(ts time.Time) {
	source.histogram.Observe(float64(time.Since(ts)))
	atomic.AddInt64(&source.count, 1)
}

func (source *prometheusTimer) Count() int64 {
	return atomic.LoadInt64(&source.count)
}

type prometheusCounter struct {
	counter prometheus.Counter
	count   int64
}

func (source *prometheusCounter) Inc() {
	source.counter.Inc()
	atomic.AddInt64(&source.count, 1)
}

func (source *prometheusCounter) Count() int64 {
	return atomic.LoadInt64(&source.count)
}

func getPrometheusMetricName(path []string) string {
	return strings.Join(path, "_")
}
