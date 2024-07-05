package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/moira-alert/moira/clock"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/events"
	"github.com/moira-alert/moira/notifier/notifications"
	"github.com/moira-alert/moira/notifier/selfstate"
	_ "go.uber.org/automaxprocs"
)

const serviceName = "notifier"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/notifier.yml", "path to config file")
	printVersion           = flag.Bool("version", false, "Print current version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira notifier bin version.
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

	logger, err = logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, serviceName, config.Logger.LogPrettyFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}
	defer logger.Info().
		String("moira_version", MoiraVersion).
		Msg("Moira Notifier stopped.")

	telemetry, err := cmd.ConfigureTelemetry(logger, config.Telemetry, serviceName)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Can not configure telemetry")
	}
	defer telemetry.Stop()

	databaseSettings := config.Redis.GetSettings()
	notificationHistorySettings := config.NotificationHistory.GetSettings()
	notificationSettings := config.Notification.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings, notificationHistorySettings, notificationSettings, redis.Notifier)

	metricSourceProvider, err := cmd.InitMetricSources(config.Remotes, database, logger)
	if err != nil {
		logger.Fatal().
			Error(err).
			Msg("Failed to initialize metric sources")
	}

	// Initialize the image store
	imageStoreMap := cmd.InitImageStores(config.ImageStores, logger)

	notifierConfig := config.Notifier.getSettings(logger)

	systemClock := clock.NewSystemClock()

	notifierMetrics := metrics.ConfigureNotifierMetrics(telemetry.Metrics, serviceName)
	sender := notifier.NewNotifier(
		database,
		logger,
		notifierConfig,
		notifierMetrics,
		metricSourceProvider,
		imageStoreMap,
		systemClock,
	)

	// Register moira senders
	if err := sender.RegisterSenders(database); err != nil {
		logger.Fatal().
			Error(err).
			Msg("Can not configure senders")
	}

	// Start moira self state checker
	if config.Notifier.SelfState.getSettings().Enabled {
		selfState := selfstate.NewSelfCheckWorker(logger, database, sender, config.Notifier.SelfState.getSettings(), metrics.ConfigureHeartBeatMetrics(telemetry.Metrics))
		if err := selfState.Start(); err != nil {
			logger.Fatal().
				Error(err).
				Msg("SelfState failed")
		}
		defer stopSelfStateChecker(selfState)
	} else {
		logger.Debug().Msg("Moira Self State Monitoring disabled")
	}

	// Start moira notification fetcher
	fetchNotificationsWorker := &notifications.FetchNotificationsWorker{
		Logger:   logger,
		Database: database,
		Notifier: sender,
		Metrics:  notifierMetrics,
	}
	fetchNotificationsWorker.Start()
	defer stopNotificationsFetcher(fetchNotificationsWorker)

	// Start moira new events fetcher
	fetchEventsWorker := &events.FetchEventsWorker{
		Logger:   logger,
		Database: database,
		Scheduler: notifier.NewScheduler(
			database,
			logger,
			notifierMetrics,
			notifier.SchedulerConfig{
				ReschedulingDelay: notifierConfig.ReschedulingDelay,
			},
			systemClock),
		Metrics: notifierMetrics,
		Config:  notifierConfig,
	}
	fetchEventsWorker.Start()
	defer stopFetchEvents(fetchEventsWorker)

	logger.Info().
		String("moira_version", MoiraVersion).
		Msg("Moira Notifier Started")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info().Msg(fmt.Sprint(<-ch))
	logger.Info().Msg("Moira Notifier shutting down.")
}

func stopFetchEvents(worker *events.FetchEventsWorker) {
	if err := worker.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop events fetcher")
	}
}

func stopNotificationsFetcher(worker *notifications.FetchNotificationsWorker) {
	if err := worker.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop notifications fetcher")
	}
}

func stopSelfStateChecker(checker *selfstate.SelfCheckWorker) {
	if err := checker.Stop(); err != nil {
		logger.Error().
			Error(err).
			Msg("Failed to stop self check worker")
	}
}
