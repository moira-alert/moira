// nolint
package metrics

import (
	"time"

	goMetrics "github.com/rcrowley/go-metrics"
)

// Timer is facade for go-metrics package Timer interface
type Timer struct {
	timer goMetrics.Timer
}

func registerTimer(name string) *Timer {
	return &Timer{goMetrics.NewRegisteredTimer(name, goMetrics.DefaultRegistry)}
}

func (timer *Timer) Count() int64 {
	return timer.timer.Count()
}

func (timer *Timer) UpdateSince(time time.Time) {
	timer.timer.UpdateSince(time)
}
