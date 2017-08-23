package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/events"
	"github.com/moira-alert/moira-alert/notifier/notifications"
	"github.com/moira-alert/moira-alert/notifier/selfstate"
)

var (
	logger                 moira.Logger
	connector              *redis.DbConnector
	configFileName         = flag.String("config", "/etc/moira/config.yml", "path to config file")
	printVersion           = flag.Bool("version", false, "Print current version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	convertDb              = flag.Bool("convert", false, "Convert telegram contacts and exit")
	//Version - sets build version during build
	Version = "latest"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Printf("Moira notifier version: %s\n", Version)
		os.Exit(0)
	}

	config := getDefault()
	if *printDefaultConfigFlag {
		cmd.PrintConfig(config)
		os.Exit(0)
	}

	err := cmd.ReadConfig(*configFileName, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not read settings: %s \n", err.Error())
		os.Exit(1)
	}

	notifierConfig := config.Notifier.getSettings()
	loggerSettings := config.Logger.GetSettings(false)

	logger, err = logging.ConfigureLog(&loggerSettings, "notifier")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	notifierMetrics := metrics.ConfigureNotifierMetrics()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	metrics.Init(config.Graphite.GetSettings(), logger, "notifier")

	connector = redis.NewDatabase(logger, config.Redis.GetSettings(), databaseMetrics)
	if *convertDb {
		convertDatabase(connector)
	}

	notifier2 := notifier.NewNotifier(connector, logger, notifierConfig, notifierMetrics)

	if err := notifier2.RegisterSenders(connector, config.Front.URI); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}

	initWorkers(notifier2, &config, notifierMetrics)
}

func initWorkers(notifier2 notifier.Notifier, config *config, metric *graphite.NotifierMetrics) {
	shutdown := make(chan bool)
	var waitGroup sync.WaitGroup

	fetchEventsWorker := events.NewFetchEventWorker(connector, logger, metric)
	fetchNotificationsWorker := notifications.NewFetchNotificationsWorker(connector, logger, notifier2)

	selfState := &selfstate.SelfCheckWorker{
		Log:      logger,
		DB:       connector,
		Config:   config.Notifier.SelfState.getSettings(),
		Notifier: notifier2,
	}
	if err := selfState.Start(); err != nil {
		logger.Fatalf("SelfState failed: %v", err)
	}

	run(fetchEventsWorker, shutdown, &waitGroup)
	run(fetchNotificationsWorker, shutdown, &waitGroup)

	logger.Infof("Moira Notifier Started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	close(shutdown)

	selfState.Stop()

	waitGroup.Wait()
	connector.DeregisterBots()
	logger.Infof("Moira Notifier Stopped. Version: %s", Version)
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
