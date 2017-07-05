package go_metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

type Metric struct {
	meter goMetrics.Meter
}

func (metric *Metric) Count() int64 {
	return metric.Count()
}

func (metric *Metric) Mark(value int64) {
	metric.Mark(value)
}

func (metric *Metric) Rate1() float64 {
	return metric.Rate1()
}

func (metric *Metric) Rate5() float64 {
	return metric.Rate5()

}

func (metric *Metric) Rate15() float64 {
	return metric.Rate15()

}

func (metric *Metric) RateMean() float64 {
	return metric.RateMean()
}
