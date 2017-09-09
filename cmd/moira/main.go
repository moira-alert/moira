package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/moira-alert/moira-alert/checker/worker"
	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
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

	log, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "main")
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
		log.Fatalf("Can't start API: %v", err)
	}

	// Filter
	filterService := &FilterService{
		Config:         config.Filter.getSettings(),
		DatabaseConfig: &databaseSettings,
		LogLevel:       config.Filter.LogLevel,
		LogFile:        config.Filter.LogFile,
	}

	if err = filterService.Start(); err != nil {
		log.Fatalf("Can't start Filter: %v", err)
	}

	// Notifier
	notifierService := &NotifierService{
		Config:          config.Notifier.getSettings(),
		SelfStateConfig: config.Notifier.SelfState.getSettings(),
		DatabaseConfig:  &databaseSettings,
		LogLevel:        config.Filter.LogLevel,
		LogFile:         config.Filter.LogFile,
	}

	if err = notifierService.Start(); err != nil {
		log.Fatalf("Can't start Notifier: %v", err)
	}

	// Checker
	checkerLog, err := logging.ConfigureLog(config.Checker.LogFile, config.Checker.LogLevel, "checker")
	if err != nil {
		log.Fatalf("Can't configure logger for Checker: %v\n", err)
	}
	checkerMetrics := metrics.ConfigureCheckerMetrics("checker")
	checkerService := &worker.Checker{
		Logger:   checkerLog,
		Database: redis.NewDatabase(checkerLog, databaseSettings),
		Config:   config.Checker.getSettings(),
		Metrics:  checkerMetrics,
		Cache:    cache.New(time.Minute, time.Minute*60),
	}

	if err = checkerService.Start(); err != nil {
		log.Fatalf("Start Checker failed: %v", err)
	}

	if err = metrics.Init(config.Graphite.GetSettings()); err != nil {
		log.Error(err)
	}

	log.Infof("Moira Started (version: %s)", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Info(<-ch)

	if err := filterService.Stop(); err != nil {
		log.Errorf("Can't stop Moira FilterService: %v", err)
	}
	log.Info("Filter stopped")

	// Stop Notifier
	notifierService.Stop()
	log.Info("Notifier stopped")

	// Stop Checker
	if err := checkerService.Stop(); err != nil {
		log.Errorf("Can't stop Moira Checker: %v", err)
	}
	log.Info("Checker stopped")

	// Stop Api
	if err := apiService.Stop(); err != nil {
		log.Errorf("Can't stop Moira Api: %v", err)
	}
	log.Info("API stopped")
	log.Infof("Moira Stopped (version: %s)", Version)
}
