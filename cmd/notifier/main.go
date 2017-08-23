package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/events"
	"github.com/moira-alert/moira-alert/notifier/notifications"
	"github.com/moira-alert/moira-alert/notifier/selfstate"
)

var (
	configFileName         = flag.String("config", "/etc/moira/config.yml", "path to config file")
	printVersion           = flag.Bool("version", false, "Print current version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	convertDb              = flag.Bool("convert", false, "Convert telegram contacts and exit")

	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	Version      = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Notifier")
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

	loggerSettings := config.Logger.GetSettings()

	logger, err := logging.ConfigureLog(&loggerSettings, "notifier")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	notifierMetrics := metrics.ConfigureNotifierMetrics()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	metrics.Init(config.Graphite.GetSettings(), logger, "notifier")

	database := redis.NewDatabase(logger, config.Redis.GetSettings(), databaseMetrics)
	if *convertDb {
		convertDatabase(database)
	}

	notifierConfig := config.Notifier.getSettings()
	sender := notifier.NewNotifier(database, logger, notifierConfig, notifierMetrics)

	if err := sender.RegisterSenders(database, config.Front.URI); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}

	selfState := &selfstate.SelfCheckWorker{
		Log:      logger,
		DB:       database,
		Config:   config.Notifier.SelfState.getSettings(),
		Notifier: sender,
	}
	if err := selfState.Start(); err != nil {
		logger.Fatalf("SelfState failed: %v", err)
	}

	fetchEventsWorker := events.FetchEventsWorker{
		Logger:    logger,
		Database:  database,
		Scheduler: notifier.NewScheduler(database, logger),
		Metrics:   notifierMetrics,
	}
	fetchEventsWorker.Start()

	fetchNotificationsWorker := &notifications.FetchNotificationsWorker{
		Logger:   logger,
		Database: database,
		Notifier: sender,
	}
	fetchNotificationsWorker.Start()

	logger.Infof("Moira Notifier Started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Notifier shutting down.")

	selfState.Stop()
	fetchEventsWorker.Stop()
	fetchNotificationsWorker.Stop()

	database.DeregisterBots()
	logger.Infof("Moira Notifier Stopped. Version: %s", Version)
}
