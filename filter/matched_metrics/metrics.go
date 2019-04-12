package matchedmetrics

import (
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/metrics/graphite"
)

// MetricsMatcher make buffer of metrics and save it
type MetricsMatcher struct {
	logger        moira.Logger
	metrics       *graphite.FilterMetrics
	database      moira.Database
	cacheStorage  *filter.Storage
	cacheCapacity int
	waitGroup     *sync.WaitGroup
}

// NewMetricsMatcher creates new MetricsMatcher
func NewMetricsMatcher(metrics *graphite.FilterMetrics, logger moira.Logger, database moira.Database, cacheStorage *filter.Storage, cacheCapacity int) *MetricsMatcher {
	return &MetricsMatcher{
		metrics:       metrics,
		logger:        logger,
		database:      database,
		cacheStorage:  cacheStorage,
		cacheCapacity: cacheCapacity,
		waitGroup:     &sync.WaitGroup{},
	}
}

// Start process matched metrics from channel and save it in cache storage
func (matcher *MetricsMatcher) Start(matchedMetricsChan chan *moira.MatchedMetric) {
	matcher.waitGroup.Add(1)
	go func() {
		defer matcher.waitGroup.Done()

		for batch := range matcher.receiveBatch(matchedMetricsChan) {
			timer := time.Now()
			matcher.save(batch)
			matcher.metrics.SavingTimer.UpdateSince(timer)
		}
	}()
	matcher.logger.Infof("Moira Filter Metrics Matcher started to save %d cached metrics every %s", matcher.cacheCapacity, time.Second.Seconds())
}

func (matcher *MetricsMatcher) receiveBatch(metrics <-chan *moira.MatchedMetric) <-chan map[string]*moira.MatchedMetric {
	batchedMetrics := make(chan map[string]*moira.MatchedMetric, 1)

	go func() {
		defer close(batchedMetrics)
		batchTimer := time.NewTimer(time.Second)
		defer batchTimer.Stop()
		for {
			batch := make(map[string]*moira.MatchedMetric, matcher.cacheCapacity)
		retry:
			select {
			case metric, ok := <-metrics:
				if !ok {
					batchedMetrics <- batch
					matcher.logger.Info("Moira Filter Metrics Matcher stopped")
					return
				}
				matcher.cacheStorage.EnrichMatchedMetric(batch, metric)
				if len(batch) < matcher.cacheCapacity {
					goto retry
				}
				batchedMetrics <- batch
			case <-batchTimer.C:
				batchedMetrics <- batch
			}
			batchTimer.Reset(time.Second)
		}
	}()
	return batchedMetrics
}

// Wait waits for metric matcher instance will stop
func (matcher *MetricsMatcher) Wait() {
	matcher.waitGroup.Wait()
}

func (matcher *MetricsMatcher) save(buffer map[string]*moira.MatchedMetric) {
	if err := matcher.database.SaveMetrics(buffer); err != nil {
		matcher.logger.Errorf("Failed to save matched metrics: %s", err.Error())
	}
}
