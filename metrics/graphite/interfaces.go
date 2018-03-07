package graphite

import "time"

// MetricsMap implements meter collection abstraction
type MetricsMap interface {
	AddMetric(name, path string)
	GetMetric(name string) (Meter, bool)
}

// TimerMap implements timer collection abstraction
type TimerMap interface {
	GetOrAdd(name, graphitePath string) Timer
}

// Meter count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter interface {
	Count() int64
	Mark(int64)
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
}

// Timer capture the duration and rate of events.
type Timer interface {
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
	StdDev() float64
	Sum() int64
	Time(func())
	Update(time.Duration)
	UpdateSince(time.Time)
	Variance() float64
}

// Gauge hold an int64 value that can be set arbitrarily.
type Gauge interface {
	Update(int64)
	Value() int64
}
