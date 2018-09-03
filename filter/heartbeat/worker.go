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
	totalCount := worker.metrics.TotalMetricsReceived.Count()
	matchedCount := worker.metrics.MatchingMetricsReceived.Count()
	worker.tomb.Go(func() error {
		checkTicker := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-worker.tomb.Dying():
				worker.logger.Info("Moira Filter Heartbeat stopped")
				return nil
			case <-checkTicker.C:
				newTotalCount := worker.metrics.TotalMetricsReceived.Count()
				worker.logger.Debugf("Update total heartbeat count, old value: %v, new value: %v", totalCount, newTotalCount)
				if newTotalCount != totalCount {
					if err := worker.database.UpdateMetricsHeartbeat(); err != nil {
						worker.logger.Infof("Save total state failed: %s", err.Error())
					} else {
						totalCount = newTotalCount
					}
				}
				newMatchedCount := worker.metrics.MatchingMetricsReceived.Count()
				worker.logger.Debugf("Update matched heartbeat count, old value: %v, new value: %v", matchedCount, newMatchedCount)
				if newMatchedCount != matchedCount {
					if err := worker.database.UpdateMatchedMetricsHeartbeat(); err != nil {
						worker.logger.Infof("Save matched state failed: %s", err.Error())
					} else {
						matchedCount = newMatchedCount
					}
				}
			}
		}
	})
	worker.logger.Info("Moira Filter Heartbeat started")
}

// Stop heartbeat worker
func (worker *Worker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
