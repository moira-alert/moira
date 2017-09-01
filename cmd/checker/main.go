package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/checker/worker"
	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/patrickmn/go-cache"
	"time"
)

var (
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

	logger, err := logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, "checker")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	database := redis.NewDatabase(logger, databaseSettings, databaseMetrics)

	checkerSettings := config.Checker.getSettings()
	if triggerID != nil && *triggerID != "" {
		checkSingleTrigger(database, logger, checkerSettings)
	}

	checkerMetrics := metrics.ConfigureCheckerMetrics()
	metrics.Init(config.Graphite.GetSettings(), logger, "checker")
	checkerWorker := &worker.Checker{
		Logger:   logger,
		Database: database,
		Config:   checkerSettings,
		Metrics:  checkerMetrics,
		Cache:    cache.New(time.Minute, time.Minute*5),
	}

	checkerWorker.Start()

	logger.Infof("Moira Checker started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Checker shutting down.")
	checkerWorker.Stop()
	logger.Infof("Moira Checker stopped. Version: %s", Version)
}

func checkSingleTrigger(database moira.Database, logger moira.Logger, settings *checker.Config) {
	triggerChecker := checker.TriggerChecker{
		TriggerID: *triggerID,
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
