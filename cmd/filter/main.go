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
	matchedmetrics "github.com/moira-alert/moira/filter/matched_metrics"
	"github.com/moira-alert/moira/filter/patterns"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	"github.com/xiam/to"
	_ "go.uber.org/automaxprocs"
)

const serviceName = "filter"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/filter.yml", "path config file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira filter bin version.
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

	logger, err = logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, serviceName, config.Logger.LogPrettyFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}
	defer logger.Info().
		String("moira_version", MoiraVersion).
		Msg("Moira Filter stopped. Version")

	compatibility := config.Filter.Compatibility.toFilterCompatibility()

	telemetry, err := cmd.ConfigureTelemetry(logger, config.Telemetry, serviceName)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Can not configure telemetry")
	}
	defer telemetry.Stop()

	if config.Filter.MaxParallelMatches == 0 {
		config.Filter.MaxParallelMatches = runtime.NumCPU()
		logger.Info().
			Int("number_of_cpu", config.Filter.MaxParallelMatches).
			Msg("MaxParallelMatches is not configured, set it to the number of CPU")
	}

	filterMetrics := metrics.ConfigureFilterMetrics(telemetry.Metrics)
	database := redis.NewDatabase(logger, config.Redis.GetSettings(), redis.NotificationHistoryConfig{}, redis.NotificationConfig{}, redis.Filter)

	retentionConfigFile, err := os.Open(config.Filter.RetentionConfig)
	if err != nil {
		logger.Fatal().
			String("file_name", config.Filter.RetentionConfig).
			Error(err).
			Msg("Error open retentions file")
	}

	cacheStorage, err := filter.NewCacheStorage(logger, filterMetrics, retentionConfigFile)
	if err != nil {
		logger.Fatal().
			String("file_name", config.Filter.RetentionConfig).
			Error(err).
			Msg("Failed to initialize cache storage with given config")
	}

	filterPatternStorageCfg := config.Filter.PatternStorageCfg.toFilterPatternStorageConfig()
	patternStorage, err := filter.NewPatternStorage(filterPatternStorageCfg, database, filterMetrics, logger, compatibility)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to refresh pattern storage")
	}

	// Refresh Patterns on first init
	refreshPatternWorker := patterns.NewRefreshPatternWorker(database, filterMetrics, logger, patternStorage, to.Duration(config.Filter.PatternsUpdatePeriod))

	// Start patterns refresher
	err = refreshPatternWorker.Start()
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to refresh pattern storage")
	}
	defer stopRefreshPatternWorker(refreshPatternWorker)

	// Start Filter heartbeat
	heartbeatWorker := heartbeat.NewHeartbeatWorker(database, filterMetrics, logger)
	heartbeatWorker.Start()
	defer stopHeartbeatWorker(heartbeatWorker)

	// Start metrics listener
	listener, err := connection.NewListener(config.Filter.Listen, logger, filterMetrics)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to start listening")
	}
	lineChan := listener.Listen()

	patternMatcher := patterns.NewMatcher(logger, filterMetrics, patternStorage, to.Duration(config.Filter.DropMetricsTTL))
	metricsChan := patternMatcher.Start(config.Filter.MaxParallelMatches, lineChan)

	// Start metrics matcher
	cacheCapacity := config.Filter.CacheCapacity
	batchForcedSaveTimeout := to.Duration(config.Filter.BatchForcedSaveTimeout)
	metricsMatcher := matchedmetrics.NewMetricsMatcher(filterMetrics, logger, database, cacheStorage, cacheCapacity, batchForcedSaveTimeout)
	metricsMatcher.Start(metricsChan)
	defer metricsMatcher.Wait()  // First stop listener
	defer stopListener(listener) // Then waiting for metrics matcher handle all received events

	logger.Info().
		String("moira_version", MoiraVersion).
		Msg("Moira Filter started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	signal := fmt.Sprint(<-ch)
	logger.Info().
		String("signal", signal).
		Msg("Moira Filter shutting down.")
}

func stopListener(listener *connection.MetricsListener) {
	if err := listener.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop listener")
	}
}

func stopHeartbeatWorker(heartbeatWorker *heartbeat.Worker) {
	if err := heartbeatWorker.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop heartbeat worker")
	}
}

func stopRefreshPatternWorker(refreshPatternWorker *patterns.RefreshPatternWorker) {
	if err := refreshPatternWorker.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop refresh pattern worker")
	}
}
