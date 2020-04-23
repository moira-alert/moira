package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/events"
	"github.com/moira-alert/moira/notifier/notifications"
	"github.com/moira-alert/moira/notifier/selfstate"
)

const serviceName = "notifier"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/notifier.yml", "path to config file")
	printVersion           = flag.Bool("version", false, "Print current version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira notifier bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Notifier")
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
	defer logger.Infof("Moira Notifier stopped. Version: %s", MoiraVersion)

	telemetry, err := cmd.ConfigureTelemetry(logger, config.Telemetry, serviceName)
	if err != nil {
		logger.Fatalf("Can not configure telemetry: %s", err.Error())
	}
	defer telemetry.Stop()

	notifierMetrics := metrics.ConfigureNotifierMetrics(telemetry.Metrics, serviceName)
	databaseSettings := config.Redis.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings, redis.Notifier)

	localSource := local.Create(database)
	remoteConfig := config.Remote.GetRemoteSourceSettings()
	remoteSource := remote.Create(remoteConfig)
	metricSourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

	// Initialize the image store
	imageStoreMap := cmd.InitImageStores(config.ImageStores, logger)

	notifierConfig := config.Notifier.getSettings(logger)

	sender := notifier.NewNotifier(database, logger, notifierConfig, notifierMetrics, metricSourceProvider, imageStoreMap)

	// Register moira senders
	if err := sender.RegisterSenders(database); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}

	// Start moira self state checker
	selfState := &selfstate.SelfCheckWorker{
		Logger:   logger,
		Database: database,
		Config:   config.Notifier.SelfState.getSettings(),
		Notifier: sender,
	}
	if err := selfState.Start(); err != nil {
		logger.Fatalf("SelfState failed: %v", err)
	}
	defer stopSelfStateChecker(selfState)

	// Start moira notification fetcher
	fetchNotificationsWorker := &notifications.FetchNotificationsWorker{
		Logger:   logger,
		Database: database,
		Notifier: sender,
	}
	fetchNotificationsWorker.Start()
	defer stopNotificationsFetcher(fetchNotificationsWorker)

	// Start moira new events fetcher
	fetchEventsWorker := &events.FetchEventsWorker{
		Logger:    logger,
		Database:  database,
		Scheduler: notifier.NewScheduler(database, logger, notifierMetrics),
		Metrics:   notifierMetrics,
	}
	fetchEventsWorker.Start()
	defer stopFetchEvents(fetchEventsWorker)

	logger.Infof("Moira Notifier Started. Version: %s", MoiraVersion)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Notifier shutting down.")
}

func stopFetchEvents(worker *events.FetchEventsWorker) {
	if err := worker.Stop(); err != nil {
		logger.Errorf("Failed to stop events fetcher: %v", err)
	}
}

func stopNotificationsFetcher(worker *notifications.FetchNotificationsWorker) {
	if err := worker.Stop(); err != nil {
		logger.Errorf("Failed to stop notifications fetcher: %v", err)
	}
}

func stopSelfStateChecker(checker *selfstate.SelfCheckWorker) {
	if err := checker.Stop(); err != nil {
		logger.Errorf("Failed to stop self check worker: %v", err)
	}
}
