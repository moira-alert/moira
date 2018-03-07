// nolint
package metrics

import (
	"time"

	goMetrics "github.com/rcrowley/go-metrics"
)

// Timer is facade for go-metrics package Meter struct
type Timer struct {
	timer goMetrics.Timer
}

func registerTimer(name string) *Timer {
	return &Timer{goMetrics.NewRegisteredTimer(name, goMetrics.DefaultRegistry)}
}

func (timer *Timer) Count() int64 {
	return timer.timer.Count()
}

func (timer *Timer) Max() int64 {
	return timer.timer.Max()
}

func (timer *Timer) Mean() float64 {
	return timer.timer.Mean()
}

func (timer *Timer) Min() int64 {
	return timer.timer.Min()
}

func (timer *Timer) Percentile(p float64) float64 {
	return timer.timer.Percentile(p)
}

func (timer *Timer) Percentiles(p []float64) []float64 {
	return timer.timer.Percentiles(p)
}

func (timer *Timer) Rate1() float64 {
	return timer.timer.Rate1()
}

func (timer *Timer) Rate5() float64 {
	return timer.timer.Rate5()
}

func (timer *Timer) Rate15() float64 {
	return timer.timer.Rate15()
}

func (timer *Timer) RateMean() float64 {
	return timer.timer.RateMean()
}

func (timer *Timer) StdDev() float64 {
	return timer.timer.StdDev()
}

func (timer *Timer) Sum() int64 {
	return timer.timer.Sum()
}

func (timer *Timer) Time(f func()) {
	timer.timer.Time(f)
}

func (timer *Timer) Update(time time.Duration) {
	timer.timer.Update(time)
}

func (timer *Timer) UpdateSince(time time.Time) {
	timer.timer.UpdateSince(time)
}

func (timer *Timer) Variance() float64 {
	return timer.timer.Variance()
}
