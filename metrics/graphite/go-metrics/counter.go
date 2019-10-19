package metrics

import (
	goMetrics "github.com/rcrowley/go-metrics"
)

// Counter is facade for go-metrics package counter interface
type Counter struct {
	counter goMetrics.Counter
}

func registerCounter(name string) *Counter {
	return &Counter{goMetrics.NewRegisteredCounter(name, goMetrics.DefaultRegistry)}
}

func (counter *Counter) Clear() {
	counter.counter.Clear()
}

func (counter *Counter) Count() int64 {
	return counter.counter.Count()
}

func (counter *Counter) Dec(val int64) {
	counter.counter.Dec(val)
}

func (counter *Counter) Inc(val int64) {
	counter.counter.Inc(val)
}
