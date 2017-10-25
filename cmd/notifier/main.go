package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/events"
	"github.com/moira-alert/moira/notifier/notifications"
	"github.com/moira-alert/moira/notifier/selfstate"
)

var serviceName = "notifier"

var (
	configFileName         = flag.String("config", "/etc/moira/config.yml", "path to config file")
	printVersion           = flag.Bool("version", false, "Print current version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	convertDb              = flag.Bool("convert", false, "Convert telegram contacts and exit")
)

// Moira notifier bin version
var (
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

	logger, err := logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	notifierMetrics := metrics.ConfigureNotifierMetrics(serviceName)
	if err = metrics.Init(config.Graphite.GetSettings()); err != nil {
		logger.Error(err)
	}

	database := redis.NewDatabase(logger, config.Redis.GetSettings())
	if *convertDb {
		convertDatabase(database)
	}

	notifierConfig := config.Notifier.getSettings()
	sender := notifier.NewNotifier(database, logger, notifierConfig, notifierMetrics)

	if err := sender.RegisterSenders(database); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}
	defer database.DeregisterBots()

	selfState := &selfstate.SelfCheckWorker{
		Log:      logger,
		DB:       database,
		Config:   config.Notifier.SelfState.getSettings(),
		Notifier: sender,
	}
	if err := selfState.Start(); err != nil {
		logger.Fatalf("SelfState failed: %v", err)
	}
	defer selfState.Stop()

	fetchEventsWorker := events.FetchEventsWorker{
		Logger:    logger,
		Database:  database,
		Scheduler: notifier.NewScheduler(database, logger, notifierMetrics),
		Metrics:   notifierMetrics,
	}
	fetchEventsWorker.Start()
	defer fetchEventsWorker.Stop()

	fetchNotificationsWorker := &notifications.FetchNotificationsWorker{
		Logger:   logger,
		Database: database,
		Notifier: sender,
	}
	fetchNotificationsWorker.Start()
	defer fetchEventsWorker.Stop()

	logger.Infof("Moira Notifier Started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Notifier shutting down.")
	defer logger.Infof("Moira Notifier Stopped. Version: %s", Version)
}
