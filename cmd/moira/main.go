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
	"github.com/moira-alert/moira/checker/worker"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
)

var (
	configFileName         = flag.String("config", "moira.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira version
var (
	Version   = "unknown"
	GitHash   = "unknown"
	GoVersion = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira - alerting system based on graphite data")
		fmt.Println("Version:", Version)
		fmt.Println("Git Commit:", GitHash)
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
		fmt.Fprintf(os.Stderr, "Can't read settings: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "main")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()

	// API
	apiService := &APIService{
		Config:         config.API.getSettings(),
		DatabaseConfig: &databaseSettings,
		LogLevel:       config.API.LogLevel,
		LogFile:        config.API.LogFile,
	}

	if err = apiService.Start(); err != nil {
		logger.Fatalf("Can't start API: %v", err)
	}
	defer stopAPI(logger, apiService)

	// Filter
	filterService := &FilterService{
		Config:         config.Filter.getSettings(),
		DatabaseConfig: &databaseSettings,
		LogLevel:       config.Filter.LogLevel,
		LogFile:        config.Filter.LogFile,
	}

	if err = filterService.Start(); err != nil {
		logger.Fatalf("Can't start Filter: %v", err)
	}
	defer stopFilter(logger, filterService)

	// Notifier
	notifierService := &NotifierService{
		Config:          config.Notifier.getSettings(logger),
		SelfStateConfig: config.Notifier.SelfState.getSettings(),
		DatabaseConfig:  &databaseSettings,
		LogLevel:        config.Filter.LogLevel,
		LogFile:         config.Filter.LogFile,
	}

	if err = notifierService.Start(); err != nil {
		logger.Fatalf("Can't start Notifier: %v", err)
	}
	defer stopNotifier(logger, notifierService)

	// Checker
	checkerLog, err := logging.ConfigureLog(config.Checker.LogFile, config.Checker.LogLevel, "checker")
	if err != nil {
		logger.Fatalf("Can't configure logger for Checker: %v\n", err)
	}
	checkerMetrics := metrics.ConfigureCheckerMetrics("checker")
	checkerService := &worker.Checker{
		Logger:   checkerLog,
		Database: redis.NewDatabase(checkerLog, databaseSettings),
		Config:   config.Checker.getSettings(),
		Metrics:  checkerMetrics,
		Cache:    cache.New(time.Minute, time.Minute*60),
	}
	defer stopChecker(logger, checkerService)

	if err = checkerService.Start(); err != nil {
		logger.Fatalf("Start Checker failed: %v", err)
	}

	if err = metrics.Init(config.Graphite.GetSettings()); err != nil {
		logger.Error(err)
	}

	logger.Infof("Moira Started (version: %s)", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(<-ch)
	logger.Infof("Moira Stopped (version: %s)", Version)
}

func stopAPI(logger moira.Logger, service *APIService) {
	if err := service.Stop(); err != nil {
		logger.Errorf("Can't stop Moira Api: %v", err)
	}
	logger.Info("API stopped")
}

func stopNotifier(logger moira.Logger, service *NotifierService) {
	service.Stop()
	logger.Info("Notifier stopped")
}

func stopChecker(logger moira.Logger, service *worker.Checker) {
	if err := service.Stop(); err != nil {
		logger.Errorf("Can't stop Moira Checker: %v", err)
	}
	logger.Info("Checker stopped")
}

func stopFilter(logger moira.Logger, service *FilterService) {
	if err := service.Stop(); err != nil {
		logger.Errorf("Can't stop Moira Filter: %v", err)
	}
	logger.Info("Filter stopped")
}
