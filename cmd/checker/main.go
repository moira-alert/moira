package main

import (
	"flag"
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/checker/worker"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var (
	configFileName = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
	printVersion   = flag.Bool("version", false, "Print version and exit")
	verbosityLog   = flag.Bool("-v", false, "Verbosity log")
	triggerId      = flag.String("t", "", "Check single trigger by id and exit")
	//Version - sets build version during build
	Version = "latest"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Printf("Moira Cache version: %s\n", Version)
		os.Exit(0)
	}

	config, err := readSettings(*configFileName)
	if err != nil {
		fmt.Printf("Can not read settings: %s \n", err.Error())
		os.Exit(1)
	}

	loggerSettings := config.Checker.getLoggerSettings(verbosityLog)

	logger, err := logging.ConfigureLog(&loggerSettings, "checker")
	if err != nil {
		fmt.Printf("Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	databaseSettings := config.Redis.getSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	database := redis.NewDatabase(logger, databaseSettings, databaseMetrics)

	checkerSettings := config.Checker.getSettings()
	if triggerId != nil && *triggerId != "" {
		checkSingleTrigger(database, logger, checkerSettings)
	}

	checkerMetrics := metrics.ConfigureCheckerMetrics()
	masterWorker := &worker.Worker{
		Logger:   logger,
		Database: database,
		Config:   checkerSettings,
		Metrics:  checkerMetrics,
	}

	masterWorker.Start()

	logger.Infof("Moira Checker started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Checker shutting down.")
	masterWorker.Stop()
	logger.Infof("Moira Checker stopped. Version: %s", Version)
}

func checkSingleTrigger(database moira.Database, logger moira.Logger, settings *checker.Config) {
	triggerChecker := checker.TriggerChecker{
		TriggerId: *triggerId,
		Database:  database,
		Logger:    logger,
		Config:    settings,
	}

	err := triggerChecker.InitTriggerChecker()
	if err != nil {
		logger.Errorf("Failed initialize trigger checker: %s", err.Error())
		os.Exit(1)
	}
	if err = triggerChecker.Check(); err != nil {
		logger.Errorf("Failed check trigger: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getCheckerLogFile(configLogFile string, checkerNumber int) string {
	if configLogFile == "" || configLogFile == "stdout" {
		return "stdout"
	}
	loggerFileName := strings.Split(configLogFile, ".")[0]
	return fmt.Sprintf("%s-{%v}.log", loggerFileName, checkerNumber)
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
