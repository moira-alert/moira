package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// Gauge is facade for go-metrics package Gauge struct
type Gauge struct {
	gauge goMetrics.Gauge
}

func (gauge *Gauge) Update(v int64) {
	gauge.gauge.Update(v)
}

func (gauge *Gauge) Value() int64 {
	return gauge.gauge.Value()
}
