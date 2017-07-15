package patterns

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
	"time"
)

type RefreshPatternWorker struct {
	database       moira.Database
	logger         moira.Logger
	metrics        *graphite.CacheMetrics
	patternStorage *cache.PatternStorage
}

func NewRefreshPatternWorker(database moira.Database, metrics *graphite.CacheMetrics, logger moira.Logger, patternStorage *cache.PatternStorage) *RefreshPatternWorker {
	return &RefreshPatternWorker{
		database:       database,
		metrics:        metrics,
		logger:         logger,
		patternStorage: patternStorage,
	}
}

func (worker *RefreshPatternWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	worker.logger.Infof("Start Moira Cache pattern updater")
	for {
		select {
		case <-shutdown:
			worker.logger.Infof("Stop Moira Cache pattern updater")
			return
		case <-time.After(time.Second):
			timer := time.Now()
			err := worker.patternStorage.RefreshTree()
			if err != nil {
				worker.logger.Errorf("pattern refresh failed: %s", err.Error())
			}
			worker.metrics.BuildTreeTimer.UpdateSince(timer)
		}
	}
}
