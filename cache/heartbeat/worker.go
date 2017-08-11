package heartbeat

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
	"time"
)

//Worker is heartbeat worker realization
type Worker struct {
	database moira.Database
	metrics  *graphite.CacheMetrics
	logger   moira.Logger
}

//NewHeartbeatWorker creates new worker
func NewHeartbeatWorker(database moira.Database, metrics *graphite.CacheMetrics, logger moira.Logger) *Worker {
	return &Worker{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
}

//Run every 5 second takes TotalMetricsReceived metrics and save it to database, for self-checking
func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	count := worker.metrics.TotalMetricsReceived.Count()

	worker.logger.Infof("Start Moira Cache Heartbeat")
	for {
		select {
		case <-shutdown:
			worker.logger.Infof("Stop Moira Cache Heartbeat")
			return
		case <-time.After(time.Second * 5):
			newCount := worker.metrics.TotalMetricsReceived.Count()
			if newCount != count {
				if err := worker.database.UpdateMetricsHeartbeat(); err != nil {
					worker.logger.Infof("Save state failed: %s", err.Error())
				} else {
					count = newCount
				}
			}
		}
	}
}
