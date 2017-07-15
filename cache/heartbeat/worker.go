package heartbeat

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"log"
	"sync"
	"time"
)

type HeartbeatWorker struct {
	database moira.Database
	metrics  *graphite.CacheMetrics
	logger   moira.Logger
}

func NewHeartbeatWorker(database moira.Database, metrics *graphite.CacheMetrics, logger moira.Logger) *HeartbeatWorker {
	return &HeartbeatWorker{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
}

func (worker *HeartbeatWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
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
					log.Printf("Save state failed: %s", err.Error())
				} else {
					count = newCount
				}
			}
		}
	}
}
