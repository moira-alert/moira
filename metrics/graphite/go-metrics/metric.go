// nolint
package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// Meter is facade for go-metrics package Meter struct
type Meter struct {
	meter goMetrics.Meter
}

func newRegisteredMeter(name string) *Meter {
	return &Meter{goMetrics.NewRegisteredMeter(name, goMetrics.DefaultRegistry)}
}

func (metric *Meter) Count() int64 {
	return metric.meter.Count()
}

func (metric *Meter) Mark(value int64) {
	metric.meter.Mark(value)
}

func (metric *Meter) Rate1() float64 {
	return metric.meter.Rate1()
}

func (metric *Meter) Rate5() float64 {
	return metric.meter.Rate5()

}

func (metric *Meter) Rate15() float64 {
	return metric.meter.Rate15()

}

func (metric *Meter) RateMean() float64 {
	return metric.meter.RateMean()
}
