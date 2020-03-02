package matchedmetrics

import (
	"sync"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/filter"
	"github.com/moira-alert/moira/internal/metrics"
)

// MetricsMatcher make buffer of metrics and save it
type MetricsMatcher struct {
	logger        moira2.Logger
	metrics       *metrics.FilterMetrics
	database      moira2.Database
	cacheStorage  *filter.Storage
	cacheCapacity int
	waitGroup     *sync.WaitGroup
}

// NewMetricsMatcher creates new MetricsMatcher
func NewMetricsMatcher(metrics *metrics.FilterMetrics, logger moira2.Logger, database moira2.Database, cacheStorage *filter.Storage, cacheCapacity int) *MetricsMatcher {
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
func (matcher *MetricsMatcher) Start(matchedMetricsChan chan *moira2.MatchedMetric) {
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

func (matcher *MetricsMatcher) receiveBatch(metrics <-chan *moira2.MatchedMetric) <-chan map[string]*moira2.MatchedMetric {
	batchedMetrics := make(chan map[string]*moira2.MatchedMetric, 1)

	go func() {
		defer close(batchedMetrics)
		batchTimer := time.NewTimer(time.Second)
		defer batchTimer.Stop()
		for {
			batch := make(map[string]*moira2.MatchedMetric, matcher.cacheCapacity)
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

func (matcher *MetricsMatcher) save(buffer map[string]*moira2.MatchedMetric) {
	if err := matcher.database.SaveMetrics(buffer); err != nil {
		matcher.logger.Errorf("Failed to save matched metrics: %s", err.Error())
	}
}
