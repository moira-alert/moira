package main

import (
	"flag"
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/handler"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var (
	configFileName = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
	printVersion   = flag.Bool("version", false, "Print version and exit")
	verbosityLog   = flag.Bool("-v", false, "Verbosity log")
	//Version - sets build version during build
	Version  = "latest"
	database moira.Database
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

	loggerSettings := config.Api.getLoggerSettings(verbosityLog)

	logger, err := logging.ConfigureLog(&loggerSettings, "api")
	if err != nil {
		fmt.Printf("Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	databaseSettings := config.Redis.getSettings()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	database = redis.NewDatabase(logger, databaseSettings, databaseMetrics)

	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			logger.Error("msg", "Error starting profiling", "error", err)
		}
	}()

	httpHandler := handler.NewHandler(database, logger)

	listeningAddress := fmt.Sprintf("%s:%s", config.Api.Address, config.Api.Port)
	logger.Infof("Start listening by address: [%s]", listeningAddress)
	http.ListenAndServe(listeningAddress, httpHandler)
	logger.Infof("Stop Moira api")
}
