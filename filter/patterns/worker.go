package patterns

import (
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/metrics/graphite"
)

// RefreshPatternWorker realization
type RefreshPatternWorker struct {
	database       moira.Database
	logger         moira.Logger
	metrics        *graphite.FilterMetrics
	patternStorage *filter.PatternStorage
	tomb           tomb.Tomb
}

// NewRefreshPatternWorker creates new RefreshPatternWorker
func NewRefreshPatternWorker(database moira.Database, metrics *graphite.FilterMetrics, logger moira.Logger, patternStorage *filter.PatternStorage) *RefreshPatternWorker {
	return &RefreshPatternWorker{
		database:       database,
		metrics:        metrics,
		logger:         logger,
		patternStorage: patternStorage,
	}
}

// Start process to refresh pattern tree every second
func (worker *RefreshPatternWorker) Start() error {
	err := worker.patternStorage.RefreshTree()
	if err != nil {
		worker.logger.Errorf("pattern refresh failed: %s", err.Error())
		return err
	}

	worker.tomb.Go(func() error {
		checkTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-worker.tomb.Dying():
				worker.logger.Info("Moira Filter Pattern Updater stopped")
				return nil
			case <-checkTicker.C:
				timer := time.Now()
				err := worker.patternStorage.RefreshTree()
				if err != nil {
					worker.logger.Errorf("Pattern refresh failed: %s", err.Error())
				}
				worker.metrics.BuildTreeTimer.UpdateSince(timer)
			}
		}
	})
	worker.logger.Info("Moira Filter Pattern Updater started")
	return nil
}

// Stop stops update pattern tree
func (worker *RefreshPatternWorker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}
