package matchedmetrics

import (
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/metrics/graphite"
	"runtime"
)

// MetricsMatcher make buffer of metrics and save it
type MetricsMatcher struct {
	logger       moira.Logger
	metrics      *graphite.FilterMetrics
	database     moira.Database
	cacheStorage *filter.Storage
	waitGroup    *sync.WaitGroup
}

// NewMetricsMatcher creates new MetricsMatcher
func NewMetricsMatcher(metrics *graphite.FilterMetrics, logger moira.Logger, database moira.Database, cacheStorage *filter.Storage) *MetricsMatcher {
	return &MetricsMatcher{
		metrics:      metrics,
		logger:       logger,
		database:     database,
		cacheStorage: cacheStorage,
		waitGroup:    &sync.WaitGroup{},
	}
}

// Start process matched metrics from channel and save it in cache storage
func (matcher *MetricsMatcher) Start(channel chan *moira.MatchedMetric) {
	numCPU := runtime.NumCPU()
	matcher.waitGroup.Add(numCPU)
	for i := 0; i < numCPU; i++ {
		go matcher.startSaver(channel)
	}
	matcher.logger.Info("Moira Filter Metrics Matcher started")
}

func (matcher *MetricsMatcher) startSaver(channel chan *moira.MatchedMetric) {
	defer matcher.waitGroup.Done()
	buffer := make(map[string]*moira.MatchedMetric)
	for {
		select {
		case metric, ok := <-channel:
			if !ok {
				matcher.logger.Info("Moira Filter Metrics Matcher stopped")
				return
			}
			matcher.cacheStorage.EnrichMatchedMetric(buffer, metric)
			if len(buffer) < 1000 {
				continue
			}
		case <-time.After(time.Second):
		}
		if len(buffer) == 0 {
			continue
		}
		timer := time.Now()
		matcher.save(buffer)
		matcher.metrics.SavingTimer.UpdateSince(timer)
		buffer = make(map[string]*moira.MatchedMetric)
	}
}

// Wait waits for metric matcher instance will stop
func (matcher *MetricsMatcher) Wait() {
	matcher.waitGroup.Wait()
}

func (matcher *MetricsMatcher) save(buffer map[string]*moira.MatchedMetric) {
	if err := matcher.database.SaveMetrics(buffer); err != nil {
		matcher.logger.Infof("Failed to save value in cache storage: %s", err.Error())
	}
}
