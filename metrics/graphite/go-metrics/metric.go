//nolint
package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

//Metric is facade for go-metrics package Meter struct

type Metric struct {
	meter goMetrics.Meter
}

func (metric *Metric) Count() int64 {
	return metric.meter.Count()
}

func (metric *Metric) Mark(value int64) {
	metric.meter.Mark(value)
}

func (metric *Metric) Rate1() float64 {
	return metric.meter.Rate1()
}

func (metric *Metric) Rate5() float64 {
	return metric.meter.Rate5()

}

func (metric *Metric) Rate15() float64 {
	return metric.meter.Rate15()

}

func (metric *Metric) RateMean() float64 {
	return metric.meter.RateMean()
}
