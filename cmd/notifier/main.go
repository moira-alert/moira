package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
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
	defer logger.Infof("Moira Notifier Stopped. Version: %s", MoiraVersion)

	if config.Pprof.Listen != "" {
		logger.Infof("Starting pprof server at: [%s]", config.Pprof.Listen)
		cmd.StartProfiling(logger, config.Pprof)
	}

	notifierMetrics := metrics.ConfigureNotifierMetrics(serviceName)

	graphiteSettings := config.Graphite.GetSettings()
	if err = metrics.Init(graphiteSettings, serviceName); err != nil {
		logger.Error(err)
	}

	databaseSettings := config.Redis.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings)

	notifierConfig := config.Notifier.getSettings(logger)
	sender := notifier.NewNotifier(database, logger, notifierConfig, notifierMetrics)

	// Register moira senders
	if err := sender.RegisterSenders(database); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}
	defer database.DeregisterBots()

	// Start moira self state checker
	selfState := &selfstate.SelfCheckWorker{
		Log:      logger,
		DB:       database,
		Config:   config.Notifier.SelfState.getSettings(),
		Notifier: sender,
	}
	if err := selfState.Start(); err != nil {
		logger.Fatalf("SelfState failed: %v", err)
	}
	defer stopSelfStateChecker(selfState)

	if err := reconvertSubscriptions(database, logger); err != nil {
		logger.Fatalf("Can not reconvert subscriptions: %s", err.Error())
	}

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

// reconvertSubscriptions iterates over existing subscriptions and replaces pseudo-tags with corresponding fields
// WARNING: This method must be removed after 2.3 release
func reconvertSubscriptions(database moira.Database, logger moira.Logger) error {
	allTags, err := database.GetTagNames()
	if err != nil {
		return err
	}
	tagSubscriptions, err := database.GetTagsSubscriptions(allTags)
	if err != nil {
		return err
	}
	converted := 0
	for _, subscription := range tagSubscriptions {
		isConverted := false
		for tagInd, tag := range subscription.Tags {
			switch tag {
			case "ERROR":
				logger.Debugf("Managing subscription %s (tags: %s) to ignore warnings", subscription.ID, strings.Join(subscription.Tags, ", "))
				subscription.IgnoreWarnings = true
				isConverted = true
				subscription.Tags = append(subscription.Tags[:tagInd], subscription.Tags[tagInd+1:]...)
			case "DEGRADATION", "HIGH DEGRADATION":
				logger.Debugf("Managing subscription %s (tags: %s) to ignore recoverings", subscription.ID, strings.Join(subscription.Tags, ", "))
				subscription.IgnoreRecoverings = true
				isConverted = true
				subscription.Tags = append(subscription.Tags[:tagInd], subscription.Tags[tagInd+1:]...)
			}
		}
		if isConverted {
			database.SaveSubscription(subscription)
		}
	}
	if converted > 0 {
		logger.Infof("Successfully converted %d pseudo-tagged subscriptions into ignore-typed subscriptions")
	}
	return nil
}
