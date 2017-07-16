package atomic

import (
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
	"time"
)

//MetricsWorker process atomic metrics data
type MetricsWorker struct {
	metrics graphite.AtomicMetrics
}

//NewAtomicMetricsWorker creates new MetricsWorker
func NewAtomicMetricsWorker(metrics graphite.AtomicMetrics) *MetricsWorker {
	return &MetricsWorker{metrics}
}

//Run every second updates atomic metrics
func (worker *MetricsWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
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
