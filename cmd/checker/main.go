package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/moira-alert/moira/checker/worker"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/prometheus"
	"github.com/moira-alert/moira/metric_source/remote"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	_ "go.uber.org/automaxprocs"
)

const serviceName = "checker"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/checker.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	triggerID              = flag.String("t", "", "Check single trigger by id and exit")
)

// Moira checker bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Checker")
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
		Msg("Moira Checker stopped")

	telemetry, err := cmd.ConfigureTelemetry(logger, config.Telemetry, serviceName)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Can not configure telemetry")
	}
	defer telemetry.Stop()

	logger.Info().Msg("Debug: checker started")

	databaseSettings := config.Redis.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings, redis.NotificationHistoryConfig{}, redis.NotificationConfig{}, redis.Checker)

	metricSourceProvider, err := initMetricSources(config, database, logger)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to initialize metric sources")
	}

	checkerMetrics := metrics.ConfigureCheckerMetrics(telemetry.Metrics, clusterKeyList(metricSourceProvider))
	checkerSettings := config.Checker.getSettings(logger)

	if triggerID != nil && *triggerID != "" {
		checkSingleTrigger(database, checkerMetrics, checkerSettings, metricSourceProvider)
	}

	checkerWorker := &worker.Checker{
		Logger:   logger,
		Database: database,
		Config:   checkerSettings,
		/// RemoteConfig:      remoteConfig,
		/// PrometheusConfig:  prometheusConfig,
		SourceProvider: metricSourceProvider,
		Metrics:        checkerMetrics,
		/// TriggerCache:      cache.New(checkerSettings.CheckInterval, time.Minute*60), //nolint
		/// LazyTriggersCache: cache.New(time.Minute*10, time.Minute*60),                //nolint
		/// PatternCache:      cache.New(checkerSettings.CheckInterval, time.Minute*60), //nolint
	}
	err = checkerWorker.Start()
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to start worker check")
	}
	defer stopChecker(checkerWorker)

	logger.Info().
		String("moira_version", MoiraVersion).
		Msg("Moira Checker started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	signal := fmt.Sprint(<-ch)
	logger.Info().
		String("signal", signal).
		Msg("Moira Checker shutting down.")
}

func checkSingleTrigger(database moira.Database, metrics *metrics.CheckerMetrics, settings *checker.Config, sourceProvider *metricSource.SourceProvider) {
	triggerChecker, err := checker.MakeTriggerChecker(*triggerID, database, logger, settings, sourceProvider, metrics)
	logger.String(moira.LogFieldNameTriggerID, *triggerID)
	if err != nil {
		logger.Error().
			Error(err).
			Msg("Failed initialize trigger checker")
		os.Exit(1)
	}
	if err = triggerChecker.Check(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed check trigger")
		os.Exit(1)
	}
	os.Exit(0)
}

func stopChecker(service *worker.Checker) {
	if err := service.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to Stop Moira Checker")
	}
}

func initMetricSources(config config, database moira.Database, logger moira.Logger) (*metricSource.SourceProvider, error) {
	provider := metricSource.CreateMetricSourceProvider()
	provider.RegisterSource(moira.MakeClusterKey(moira.GraphiteLocal, "default"), local.Create(database))

	for _, graphite := range config.Remotes.Graphite {
		config := graphite.GetRemoteSourceSettings()
		source := remote.Create(config)
		provider.RegisterSource(moira.MakeClusterKey(moira.GraphiteRemote, graphite.ClusterId), source)
	}

	for _, prom := range config.Remotes.Prometheus {
		config := prom.GetPrometheusSourceSettings()
		source, err := prometheus.Create(config, logger)
		if err != nil {
			return nil, err
		}
		provider.RegisterSource(moira.MakeClusterKey(moira.GraphiteRemote, prom.ClusterId), source)
	}

	return provider, nil
}

func clusterKeyList(provider *metricSource.SourceProvider) []moira.ClusterKey {
	keys := make([]moira.ClusterKey, 0, len(provider.GetAllSources()))
	for ck := range provider.GetAllSources() {
		keys = append(keys, ck)
	}
	return keys
}
