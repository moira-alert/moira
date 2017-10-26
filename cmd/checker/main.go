package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/checker/worker"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
)

const serviceName = "checker"

var (
	logger                 moira.Logger
	configFileName         = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
	triggerID              = flag.String("t", "", "Check single trigger by id and exit")
)

// Moira checker bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	Version      = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Checker")
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
	defer logger.Infof("Moira Checker stopped. Version: %s", Version)

	databaseSettings := config.Redis.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings)

	checkerMetrics := metrics.ConfigureCheckerMetrics(serviceName)
	if err = metrics.Init(config.Graphite.GetSettings()); err != nil {
		logger.Error(err)
	}

	checkerSettings := config.Checker.getSettings()
	if triggerID != nil && *triggerID != "" {
		checkSingleTrigger(database, checkerMetrics, checkerSettings)
	}
	checkerWorker := &worker.Checker{
		Logger:   logger,
		Database: database,
		Config:   checkerSettings,
		Metrics:  checkerMetrics,
		Cache:    cache.New(time.Minute, time.Minute*60),
	}
	checkerWorker.Start()
	defer stopChecker(checkerWorker)

	logger.Infof("Moira Checker started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Checker shutting down.")
}

func checkSingleTrigger(database moira.Database, metrics *graphite.CheckerMetrics, settings *checker.Config) {
	triggerChecker := checker.TriggerChecker{
		TriggerID: *triggerID,
		Database:  database,
		Logger:    logger,
		Config:    settings,
		Metrics:   metrics,
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

func stopChecker(service *worker.Checker) {
	if err := service.Stop(); err != nil {
		logger.Errorf("Failed to Stop Moira Checker: %v", err)
	}
}
