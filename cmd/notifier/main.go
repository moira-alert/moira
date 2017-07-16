package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/moira-alert/moira-alert"
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
	logger         moira.Logger
	connector      *redis.DbConnector
	configFileName = flag.String("config", "/etc/moira/config.yml", "path to config file")
	printVersion   = flag.Bool("version", false, "Print current version and exit")
	convertDb      = flag.Bool("convert", false, "Convert telegram contacts and exit")
	//Version - sets build version during build
	Version = "latest"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Printf("Moira notifier version: %s\n", Version)
		os.Exit(0)
	}

	config, err := readSettings(*configFileName)
	if err != nil {
		fmt.Printf("Can not read settings: %s \n", err.Error())
		os.Exit(1)
	}

	notifierConfig := config.Notifier.getSettings()
	loggerSettings := config.Notifier.getLoggerSettings()

	logger, err = logging.ConfigureLog(&loggerSettings)
	if err != nil {
		fmt.Printf("Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	metric := metrics.ConfigureNotifierMetrics()
	metrics.Init(config.Graphite.getSettings(), logger)

	connector = redis.Init(logger, config.Redis.getSettings(), metric)
	if *convertDb {
		convertDatabase(connector)
	}

	notifier2 := notifier.Init(connector, logger, notifierConfig, metric)

	if err := notifier2.RegisterSenders(connector, config.Front.URI); err != nil {
		logger.Fatalf("Can not configure senders: %s", err.Error())
	}

	initWorkers(notifier2, config, metric)
}

func initWorkers(notifier2 notifier.Notifier, config *config, metric *graphite.NotifierMetrics) {
	shutdown := make(chan bool)
	var waitGroup sync.WaitGroup

	fetchEventsWorker := events.Init(connector, logger, metric)
	fetchNotificationsWorker := notifications.Init(connector, logger, notifier2)

	runSelfStateMonitorIfNeed(notifier2, config.Notifier.SelfState, shutdown, &waitGroup)
	run(fetchEventsWorker, shutdown, &waitGroup)
	run(fetchNotificationsWorker, shutdown, &waitGroup)

	logger.Infof("Moira Notifier Started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	close(shutdown)
	waitGroup.Wait()
	connector.DeregisterBots()
	logger.Infof("Moira Notifier Stopped. Version: %s", Version)
}

func runSelfStateMonitorIfNeed(notifier2 notifier.Notifier, config selfStateConfig, shutdown chan bool, waitGroup *sync.WaitGroup) {
	selfStateConfiguration := config.getSettings()
	worker, needRun := selfstate.Init(connector, logger, selfStateConfiguration, notifier2)
	if needRun {
		run(worker, shutdown, waitGroup)
	} else {
		logger.Debugf("Moira Self State Monitoring disabled")
	}
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
