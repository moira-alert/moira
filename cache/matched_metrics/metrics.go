package matchedmetrics

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
	"time"
)

//MetricsMatcherProcessor make buffer of metrics and save it
type MetricsMatcherProcessor struct {
	logger       moira.Logger
	metrics      *graphite.CacheMetrics
	database     moira.Database
	cacheStorage *cache.Storage
}

//NewMatchedMetricsProcessor creates new MetricsMatcherProcessor
func NewMatchedMetricsProcessor(metrics *graphite.CacheMetrics, logger moira.Logger, database moira.Database, cacheStorage *cache.Storage) *MetricsMatcherProcessor {
	return &MetricsMatcherProcessor{
		metrics:      metrics,
		logger:       logger,
		database:     database,
		cacheStorage: cacheStorage,
	}
}

//Run process matched metrics from channel and save it in cache
func (processor *MetricsMatcherProcessor) Run(channel chan *moira.MatchedMetric, wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := make(map[string]*moira.MatchedMetric)
	for {
		select {
		case metric, ok := <-channel:
			if !ok {
				return
			}

			processor.cacheStorage.EnrichMatchedMetric(buffer, metric)

			if len(buffer) < 10 {
				continue
			}
		case <-time.After(time.Second):
		}
		if len(buffer) == 0 {
			continue
		}
		timer := time.Now()
		processor.save(buffer)
		processor.metrics.SavingTimer.UpdateSince(timer)
		buffer = make(map[string]*moira.MatchedMetric)
	}
}

func (processor *MetricsMatcherProcessor) save(buffer map[string]*moira.MatchedMetric) {
	if err := processor.database.SaveMetrics(buffer); err != nil {
		processor.logger.Infof("Failed to save value in cache: %s", err.Error())
	}
}
