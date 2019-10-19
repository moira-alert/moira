package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// ExpDecayHistogram is facade for go-metrics package Histogram struct
// This histogram uses Exponentially Decaying Reservoir
// For more description see http://metrics.dropwizard.io/4.0.0/manual/core.html#histograms
type ExpDecayHistogram struct {
	histogram goMetrics.Histogram
}

func registerHistogram(name string) *ExpDecayHistogram {
	return &ExpDecayHistogram{goMetrics.NewRegisteredHistogram(name, goMetrics.DefaultRegistry, goMetrics.NewExpDecaySample(1028, 0.015))}
}

func (histogram *ExpDecayHistogram) Clear() {
	histogram.histogram.Clear()
}

func (histogram *ExpDecayHistogram) Count() int64 {
	return histogram.histogram.Count()
}

func (histogram *ExpDecayHistogram) Max() int64 {
	return histogram.histogram.Max()
}

func (histogram *ExpDecayHistogram) Mean() float64 {
	return histogram.histogram.Mean()
}

func (histogram *ExpDecayHistogram) Min() int64 {
	return histogram.histogram.Min()
}

func (histogram *ExpDecayHistogram) Percentile(p float64) float64 {
	return histogram.histogram.Percentile(p)
}

func (histogram *ExpDecayHistogram) Percentiles(p []float64) []float64 {
	return histogram.histogram.Percentiles(p)
}

func (histogram *ExpDecayHistogram) StdDev() float64 {
	return histogram.histogram.StdDev()
}

func (histogram *ExpDecayHistogram) Sum() int64 {
	return histogram.histogram.Sum()
}

func (histogram *ExpDecayHistogram) Update(v int64) {
	histogram.histogram.Update(v)
}

func (histogram *ExpDecayHistogram) Variance() float64 {
	return histogram.histogram.Variance()
}
