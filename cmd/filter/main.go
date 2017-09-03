package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/filter"
	"github.com/moira-alert/moira-alert/filter/connection"
	"github.com/moira-alert/moira-alert/filter/heartbeat"
	"github.com/moira-alert/moira-alert/filter/matched_metrics"
	"github.com/moira-alert/moira-alert/filter/patterns"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
)

var (
	configFileName         = flag.String("config", "/etc/moira/config.yml", "path config file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira filter bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	Version      = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Filter")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", Version)
		os.Exit(0)
	}

	config := getDefault()
	if *printDefaultConfigFlag {
		cmd.PrintConfig(config)
		os.Exit(0)
	}

	err := cmd.ReadConfig(*configFileName, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not read settings: %s\n", err.Error())
		os.Exit(1)
	}

	logger, err := logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, "filter")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	cacheMetrics := metrics.ConfigureFilterMetrics("filter")
	if err = metrics.Init(config.Graphite.GetSettings()); err != nil {
		logger.Error(err)
	}

	database := redis.NewDatabase(logger, config.Redis.GetSettings())

	retentionConfigFile, err := os.Open(config.Filter.RetentionConfig)
	if err != nil {
		logger.Fatalf("Error open retentions file [%s]: %s", config.Filter.RetentionConfig, err.Error())
	}

	cacheStorage, err := filter.NewCacheStorage(cacheMetrics, retentionConfigFile)
	if err != nil {
		logger.Fatalf("Failed to initialize cache storage with config [%s]: %s", config.Filter.RetentionConfig, err.Error())
	}

	patternStorage, err := filter.NewPatternStorage(database, cacheMetrics, logger)
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}

	refreshPatternWorker := patterns.NewRefreshPatternWorker(database, cacheMetrics, logger, patternStorage)
	heartbeatWorker := heartbeat.NewHeartbeatWorker(database, cacheMetrics, logger)

	err = refreshPatternWorker.Start()
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}
	heartbeatWorker.Start()

	listener, err := connection.NewListener(config.Filter.Listen, logger, patternStorage)
	if err != nil {
		logger.Fatalf("Failed to start listen: %s", err.Error())
	}
	metricsMatcher := matchedmetrics.NewMetricsMatcher(cacheMetrics, logger, database, cacheStorage)

	metricsChan := listener.Listen()
	var matcherWG sync.WaitGroup
	metricsMatcher.Start(metricsChan, &matcherWG)

	logger.Infof("Moira Filter started. Version: %s", MoiraVersion)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Filter shutting down.")

	listener.Stop()
	matcherWG.Wait()
	refreshPatternWorker.Stop()
	heartbeatWorker.Stop()

	logger.Infof("Moira Filter stopped. Version: %s", MoiraVersion)
}
