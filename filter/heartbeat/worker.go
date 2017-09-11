package heartbeat

import (
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite"
)

// Worker is heartbeat worker realization
type Worker struct {
	database moira.Database
	metrics  *graphite.FilterMetrics
	logger   moira.Logger
	tomb     tomb.Tomb
}

// NewHeartbeatWorker creates new worker
func NewHeartbeatWorker(database moira.Database, metrics *graphite.FilterMetrics, logger moira.Logger) *Worker {
	return &Worker{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
}

// Start every 5 second takes TotalMetricsReceived metrics and save it to database, for self-checking
func (worker *Worker) Start() {
	count := worker.metrics.TotalMetricsReceived.Count()
	worker.tomb.Go(func() error {
		checkTicker := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-worker.tomb.Dying():
				worker.logger.Info("Moira Filter heartbeat stopped")
				return nil
			case <-checkTicker.C:
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
	})
	worker.logger.Info("Moira Filter heartbeat started")
}

// Stop heartbeat worker
func (worker *Worker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
