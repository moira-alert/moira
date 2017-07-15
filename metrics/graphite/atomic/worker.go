package atomic

import (
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
	"time"
)

type AtomicMetricsWorker struct {
	metrics graphite.AtomicMetrics
}

func NewAtomicMetricsWorker(metrics graphite.AtomicMetrics) *AtomicMetricsWorker {
	return &AtomicMetricsWorker{metrics}
}

func (worker *AtomicMetricsWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-shutdown:
			return
		case <-time.After(time.Second):
			worker.metrics.UpdateMetrics()
		}
	}
}
