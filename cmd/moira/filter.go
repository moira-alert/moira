package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/filter/connection"
	"github.com/moira-alert/moira/filter/heartbeat"
	"github.com/moira-alert/moira/filter/matched_metrics"
	"github.com/moira-alert/moira/filter/patterns"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
)

// FilterService represents filter functionality of moira
type FilterService struct {
	Config         *filter.Config
	DatabaseConfig *redis.Config

	LogFile  string
	LogLevel string

	listener             *connection.MetricsListener
	matcherWG            *sync.WaitGroup
	refreshPatternWorker *patterns.RefreshPatternWorker
	heartbeatWorker      *heartbeat.Worker
}

// Start Moira Filter
func (filterService *FilterService) Start() error {
	logger, err := logging.ConfigureLog(filterService.LogFile, filterService.LogLevel, "filter")
	if err != nil {
		return fmt.Errorf("Can't configure logger for Filter: %v", err)
	}

	if !filterService.Config.Enabled {
		logger.Info("Moira Filter disabled")
		return nil
	}

	dataBase := redis.NewDatabase(logger, *filterService.DatabaseConfig)
	cacheMetrics := metrics.ConfigureFilterMetrics("filter")

	retentionConfigFile, err := os.Open(filterService.Config.RetentionConfig)
	if err != nil {
		return err
	}

	cacheStorage, err := filter.NewCacheStorage(cacheMetrics, retentionConfigFile)
	if err != nil {
		return fmt.Errorf("Failed to initialize cache storage with config [%s]: %v", filterService.Config.RetentionConfig, err)
	}

	patternStorage, err := filter.NewPatternStorage(dataBase, cacheMetrics, logger)
	if err != nil {
		return fmt.Errorf("Failed to refresh pattern storage: %s", err.Error())
	}

	filterService.refreshPatternWorker = patterns.NewRefreshPatternWorker(dataBase, cacheMetrics, logger, patternStorage)
	filterService.heartbeatWorker = heartbeat.NewHeartbeatWorker(dataBase, cacheMetrics, logger)

	if err = filterService.refreshPatternWorker.Start(); err != nil {
		return fmt.Errorf("Failed to start refresh pattern storage: %s", err.Error())
	}
	filterService.heartbeatWorker.Start()

	if filterService.listener, err = connection.NewListener(filterService.Config.Listen, logger, patternStorage); err != nil {
		return fmt.Errorf("Failed to start listen: %s", err.Error())
	}

	metricsMatcher := matchedmetrics.NewMetricsMatcher(cacheMetrics, logger, dataBase, cacheStorage)

	metricsChan := filterService.listener.Listen()
	metricsMatcher.Start(metricsChan, filterService.matcherWG)
	return nil
}

// Stop Moira Filter
func (filterService *FilterService) Stop() error {
	if err := filterService.listener.Stop(); err != nil {
		return err
	}
	filterService.matcherWG.Wait()
	if err := filterService.refreshPatternWorker.Stop(); err != nil {
		return err
	}
	if err := filterService.heartbeatWorker.Stop(); err != nil {
		return err
	}
	return nil
}
