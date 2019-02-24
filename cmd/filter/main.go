package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/filter/connection"
	"github.com/moira-alert/moira/filter/heartbeat"
	"github.com/moira-alert/moira/filter/matched_metrics"
	"github.com/moira-alert/moira/filter/patterns"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
)

const serviceName = "filter"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/filter.yml", "path config file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira filter bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Filter")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
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

	logger, err = logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}
	defer logger.Infof("Moira Filter stopped. Version: %s", MoiraVersion)

	if config.Filter.MaxParallelMatches == 0 {
		config.Filter.MaxParallelMatches = runtime.NumCPU()
		logger.Infof("MaxParallelMatches is not configured, set it to the number of CPU - %d", config.Filter.MaxParallelMatches)
	}

	if config.Pprof.Listen != "" {
		logger.Infof("Starting pprof server at: [%s]", config.Pprof.Listen)
		cmd.StartProfiling(logger, config.Pprof)
	}

	cacheMetrics := metrics.ConfigureFilterMetrics(serviceName)

	graphiteSettings := config.Graphite.GetSettings()
	if err = metrics.Init(graphiteSettings, serviceName); err != nil {
		logger.Error(err)
	}

	database := redis.NewDatabase(logger, config.Redis.GetSettings(), redis.Filter)

	retentionConfigFile, err := os.Open(config.Filter.RetentionConfig)
	if err != nil {
		logger.Fatalf("Error open retentions file [%s]: %s", config.Filter.RetentionConfig, err.Error())
	}

	cacheStorage, err := filter.NewCacheStorage(logger, cacheMetrics, retentionConfigFile)
	if err != nil {
		logger.Fatalf("Failed to initialize cache storage with config [%s]: %s", config.Filter.RetentionConfig, err.Error())
	}

	patternStorage, err := filter.NewPatternStorage(database, cacheMetrics, logger)
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}

	// Refresh Patterns on first init
	refreshPatternWorker := patterns.NewRefreshPatternWorker(database, cacheMetrics, logger, patternStorage)

	// Start patterns refresher
	err = refreshPatternWorker.Start()
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}
	defer stopRefreshPatternWorker(refreshPatternWorker)

	// Start Filter heartbeat
	heartbeatWorker := heartbeat.NewHeartbeatWorker(database, cacheMetrics, logger)
	heartbeatWorker.Start()
	defer stopHeartbeatWorker(heartbeatWorker)

	// Start metrics listener
	listener, err := connection.NewListener(config.Filter.Listen, config.Filter.Compression, logger, cacheMetrics)
	if err != nil {
		logger.Fatalf("Failed to start listen: %s", err.Error())
	}
	lineChan := listener.Listen()

	patternMatcher := patterns.NewMatcher(logger, cacheMetrics, patternStorage)
	metricsChan := patternMatcher.Start(config.Filter.MaxParallelMatches, lineChan)

	// Start metrics matcher
	cacheCapacity := config.Filter.CacheCapacity
	metricsMatcher := matchedmetrics.NewMetricsMatcher(cacheMetrics, logger, database, cacheStorage, cacheCapacity)
	metricsMatcher.Start(metricsChan)
	defer metricsMatcher.Wait()  // First stop listener
	defer stopListener(listener) // Then waiting for metrics matcher handle all received events

	logger.Infof("Moira Filter started. Version: %s", MoiraVersion)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Filter shutting down.")
}

func stopListener(listener *connection.MetricsListener) {
	if err := listener.Stop(); err != nil {
		logger.Errorf("Failed to stop listener: %v", err)
	}
}

func stopHeartbeatWorker(heartbeatWorker *heartbeat.Worker) {
	if err := heartbeatWorker.Stop(); err != nil {
		logger.Errorf("Failed to stop heartbeat worker: %v", err)
	}
}

func stopRefreshPatternWorker(refreshPatternWorker *patterns.RefreshPatternWorker) {
	if err := refreshPatternWorker.Stop(); err != nil {
		logger.Errorf("Failed to stop refresh pattern worker: %v", err)
	}
}
