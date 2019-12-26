// nolint
package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// Meter is facade for go-metrics package Meter struct
type Meter struct {
	meter goMetrics.Meter
}

func registerMeter(name string) *Meter {
	return &Meter{goMetrics.NewRegisteredMeter(name, goMetrics.DefaultRegistry)}
}

func (metric *Meter) Mark(value int64) {
	metric.meter.Mark(value)
}
