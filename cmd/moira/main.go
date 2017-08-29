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
)

var (
	configFileName         = flag.String("config", "moira.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	Version      = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Api")
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
		fmt.Fprintf(os.Stderr, "Can't read settings: %v\n", err)
		os.Exit(1)
	}

	log, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "main")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()

	// API
	apiLog, err := logging.ConfigureLog(config.API.LogFile, config.API.LogLevel, "api")
	if err != nil {
		log.Fatalf("Can't configure logger for API: %v\n", err)
	}

	apiServer := &APIServer{
		Config: config.API.getSettings(),
		DB:     redis.NewDatabase(apiLog, databaseSettings, databaseMetrics),
		Log:    apiLog,
	}

	if err := apiServer.Start(); err != nil {
		log.Fatalf("Can't start API: %v", err)
	}

	// Filter
	filterLog, err := logging.ConfigureLog(config.Filter.LogFile, config.Filter.LogLevel, "filter")
	if err != nil {
		log.Fatalf("Can't configure logger for Filter: %v\n", err)
	}

	filterServer := &Filter{
		Config: config.Filter.getSettings(),
		DB:     redis.NewDatabase(filterLog, databaseSettings, databaseMetrics),
		Log:    filterLog,
	}

	if err := filterServer.Start(); err != nil {
		log.Fatalf("Can't start Filter: %v", err)
	}

	metrics.Init(config.Graphite.GetSettings(), log, "moira")

	log.Infof("Moira Started (version: %s)", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Info(<-ch)

	if err := apiServer.Stop(); err != nil {
		log.Errorf("Can't stop API: %v", err)
	}

	if err := filterServer.Stop(); err != nil {
		log.Errorf("Can't stop Filer: %v", err)
	}

	log.Infof("Moira Stopped (version: %s)", Version)
}
