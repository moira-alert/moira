package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/cache/connection"
	"github.com/moira-alert/moira-alert/cache/heartbeat"
	"github.com/moira-alert/moira-alert/cache/matched_metrics"
	"github.com/moira-alert/moira-alert/cache/patterns"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
)

// Filter represents filter functionality of moira
type Filter struct {
	Config *cache.Config
	Log    moira.Logger
	DB     moira.Database

	listener             *connection.MetricsListener
	matcherWG            *sync.WaitGroup
	refreshPatternWorker *patterns.RefreshPatternWorker
	heartbeatWorker      *heartbeat.Worker
}

// Start Moira Filter
func (filter *Filter) Start() error {
	if !filter.Config.Enabled {
		filter.Log.Debug("Filter disabled")
		return nil
	}

	cacheMetrics := metrics.ConfigureCacheMetrics("cache")

	retentionConfigFile, err := os.Open(filter.Config.RetentionConfig)
	if err != nil {
		return err
	}

	cacheStorage, err := cache.NewCacheStorage(cacheMetrics, retentionConfigFile)
	if err != nil {
		return fmt.Errorf("Failed to initialize cache with config [%s]: %v", filter.Config.RetentionConfig, err)
	}

	patternStorage, err := cache.NewPatternStorage(filter.DB, cacheMetrics, filter.Log)
	if err != nil {
		return fmt.Errorf("Failed to refresh pattern storage: %s", err.Error())
	}

	filter.refreshPatternWorker = patterns.NewRefreshPatternWorker(filter.DB, cacheMetrics, filter.Log, patternStorage)

	filter.heartbeatWorker = heartbeat.NewHeartbeatWorker(filter.DB, cacheMetrics, filter.Log)

	if err = filter.refreshPatternWorker.Start(); err != nil {
		return fmt.Errorf("Failed to start refresh pattern storage: %s", err.Error())
	}

	filter.heartbeatWorker.Start()

	if filter.listener, err = connection.NewListener(filter.Config.Listen, filter.Log, patternStorage); err != nil {
		return fmt.Errorf("Failed to start listen: %s", err.Error())
	}

	metricsMatcher := matchedmetrics.NewMetricsMatcher(cacheMetrics, filter.Log, filter.DB, cacheStorage)

	metricsChan := filter.listener.Listen()
	metricsMatcher.Start(metricsChan, filter.matcherWG)

	return nil
}

// Stop Moira Filter
func (filter *Filter) Stop() error {
	if err := filter.listener.Stop(); err != nil {
		filter.Log.Error(err)
	}
	filter.matcherWG.Wait()
	if err := filter.refreshPatternWorker.Stop(); err != nil {
		filter.Log.Error(err)
	}
	if err := filter.heartbeatWorker.Stop(); err != nil {
		filter.Log.Error(err)
	}
	return nil
}
