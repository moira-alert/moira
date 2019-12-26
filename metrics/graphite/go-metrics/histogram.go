// nolint
package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// Histogram is facade for go-metrics package Histogram interface
type Histogram struct {
	histogram goMetrics.Histogram
}

func registerHistogram(name string) *Histogram {
	return &Histogram{goMetrics.NewRegisteredHistogram(name, goMetrics.DefaultRegistry, goMetrics.NewExpDecaySample(1028, 0.015))}
}

func (histogram *Histogram) Count() int64 {
	return histogram.histogram.Count()
}

func (histogram *Histogram) Update(v int64) {
	histogram.histogram.Update(v)
}
