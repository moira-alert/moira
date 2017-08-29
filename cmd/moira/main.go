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
	configFileName         = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
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

	// Config
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

	// Logger
	loggerSettings := config.Logger.GetSettings()

	logger, err := logging.ConfigureLog(&loggerSettings, "moira")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure log: %v\n", err)
		os.Exit(1)
	}

	// Database
	databaseSettings := config.Redis.GetSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	database := redis.NewDatabase(logger, databaseSettings, databaseMetrics)

	// Metrics
	metrics.Init(config.Graphite.GetSettings(), logger, "moira")

	// API
	apiServer := &APIServer{
		Config: config.API.getSettings(),
		DB:     database,
		Log:    logger,
	}

	if err := apiServer.Start(); err != nil {
		logger.Fatalf("Can't start API: %v", err)
	}

	// Filter
	filterServer := &Filter{
		Config: config.Cache.getSettings(),
		DB: database,
		Log: logger,
	}

	if err := filterServer.Start(); err != nil {
		logger.Fatalf("Can't start Filter: %v", err)
	}

	logger.Infof("Moira Started (version: %s)", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(<-ch)

	if err := apiServer.Stop(); err != nil {
		logger.Errorf("Can't stop API: %v", err)
	}

	if err := filterServer.Stop(); err != nil {
		logger.Errorf("Can't stop Filer: %v", err)
	}

	logger.Infof("Moira Stopped (version: %s)", Version)
}
