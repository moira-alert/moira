package heartbeat

import (
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// Worker is heartbeat worker realization
type Worker struct {
	database moira.Database
	metrics  *metrics.FilterMetrics
	logger   moira.Logger
	tomb     tomb.Tomb
}

// NewHeartbeatWorker creates new worker
func NewHeartbeatWorker(database moira.Database, metrics *metrics.FilterMetrics, logger moira.Logger) *Worker {
	return &Worker{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
}

// Start every 5 second takes TotalMetricsReceived metrics and save it to database, for self-checking
func (worker *Worker) Start() {
	worker.tomb.Go(func() error {
		count := worker.metrics.TotalMetricsReceived.Count()
		checkTicker := time.NewTicker(time.Second * 5) //nolint
		for {
			select {
			case <-worker.tomb.Dying():
				worker.logger.Info().Msg("Moira Filter Heartbeat stopped")
				return nil
			case <-checkTicker.C:
				newCount := worker.metrics.TotalMetricsReceived.Count()
				if newCount != count {
					worker.logger.Debug().
						Int64("from", count).
						Int64("to", newCount).
						Msg("Heartbeat was updated")

					if err := worker.database.UpdateMetricsHeartbeat(); err != nil {
						worker.logger.Info().
							Error(err).
							Msg("Save state failed")
					} else {
						count = newCount
					}
				}
			}
		}
	})
	worker.logger.Info().Msg("Moira Filter Heartbeat started")
}

// Stop heartbeat worker
func (worker *Worker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
