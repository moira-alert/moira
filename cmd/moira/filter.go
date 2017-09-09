package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/filter"
	"github.com/moira-alert/moira-alert/filter/connection"
	"github.com/moira-alert/moira-alert/filter/heartbeat"
	"github.com/moira-alert/moira-alert/filter/matched_metrics"
	"github.com/moira-alert/moira-alert/filter/patterns"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
)

// Filter represents filter functionality of moira
type Filter struct {
	Config *filter.Config
	Log    moira.Logger
	DB     moira.Database

	listener             *connection.MetricsListener
	matcherWG            *sync.WaitGroup
	refreshPatternWorker *patterns.RefreshPatternWorker
	heartbeatWorker      *heartbeat.Worker
}

// Start Moira Filter
func (f *Filter) Start() error {
	if !f.Config.Enabled {
		f.Log.Info("Filter disabled")
		return nil
	}

	cacheMetrics := metrics.ConfigureFilterMetrics("filter")

	retentionConfigFile, err := os.Open(f.Config.RetentionConfig)
	if err != nil {
		return err
	}

	cacheStorage, err := filter.NewCacheStorage(cacheMetrics, retentionConfigFile)
	if err != nil {
		return fmt.Errorf("Failed to initialize cache storage with config [%s]: %v", f.Config.RetentionConfig, err)
	}

	patternStorage, err := filter.NewPatternStorage(f.DB, cacheMetrics, f.Log)
	if err != nil {
		return fmt.Errorf("Failed to refresh pattern storage: %s", err.Error())
	}

	f.refreshPatternWorker = patterns.NewRefreshPatternWorker(f.DB, cacheMetrics, f.Log, patternStorage)

	f.heartbeatWorker = heartbeat.NewHeartbeatWorker(f.DB, cacheMetrics, f.Log)

	if err = f.refreshPatternWorker.Start(); err != nil {
		return fmt.Errorf("Failed to start refresh pattern storage: %s", err.Error())
	}

	f.heartbeatWorker.Start()

	if f.listener, err = connection.NewListener(f.Config.Listen, f.Log, patternStorage); err != nil {
		return fmt.Errorf("Failed to start listen: %s", err.Error())
	}

	metricsMatcher := matchedmetrics.NewMetricsMatcher(cacheMetrics, f.Log, f.DB, cacheStorage)

	metricsChan := f.listener.Listen()
	metricsMatcher.Start(metricsChan, f.matcherWG)

	return nil
}

// Stop Moira Filter
func (f *Filter) Stop() error {
	if err := f.listener.Stop(); err != nil {
		f.Log.Error(err)
	}
	f.matcherWG.Wait()
	if err := f.refreshPatternWorker.Stop(); err != nil {
		f.Log.Error(err)
	}
	if err := f.heartbeatWorker.Stop(); err != nil {
		f.Log.Error(err)
	}
	return nil
}
