package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/handler"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/index"
	"github.com/moira-alert/moira/logging/go-logging"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"
)

const serviceName = "api"

var (
	configFileName         = flag.String("config", "/etc/moira/api.yml", "Path to configuration file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")
)

// Moira api bin version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Api")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
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
		fmt.Fprintf(os.Stderr, "Can not read settings: %s\n", err.Error())
		os.Exit(1)
	}

	apiConfig := config.API.getSettings(config.Redis.MetricsTTL, config.Remote.MetricsTTL)

	logger, err := logging.ConfigureLog(config.Logger.LogFile, config.Logger.LogLevel, serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}
	defer logger.Infof("Moira API stopped. Version: %s", MoiraVersion)

	telemetry, err := cmd.ConfigureTelemetry(logger, config.Telemetry, serviceName)
	if err != nil {
		logger.Fatalf("Can not start telemetry: %s", err.Error())
	}
	defer telemetry.Stop()

	databaseSettings := config.Redis.GetSettings()
	database := redis.NewDatabase(logger, databaseSettings, redis.API)

	// Start Index right before HTTP listener. Fail if index cannot start
	searchIndex := index.NewSearchIndex(logger, database, telemetry.Metrics)
	if searchIndex == nil {
		logger.Fatalf("Failed to create search index")
	}

	err = searchIndex.Start()
	if err != nil {
		logger.Fatalf("Failed to start search index: %s", err.Error())
	}
	defer searchIndex.Stop() //nolint

	if !searchIndex.IsReady() {
		logger.Fatalf("Search index is not ready, exit")
	}

	// Start listener only after index is ready
	listener, err := net.Listen("tcp", apiConfig.Listen)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Start listening by address: [%s]", apiConfig.Listen)

	localSource := local.Create(database)
	remoteConfig := config.Remote.GetRemoteSourceSettings()
	remoteSource := remote.Create(remoteConfig)
	metricSourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

	webConfigContent, err := config.Web.getSettings(remoteConfig.Enabled)
	if err != nil {
		logger.Fatal(err)
	}

	httpHandler := handler.NewHandler(database, logger, searchIndex, apiConfig, metricSourceProvider, webConfigContent)
	server := &http.Server{
		Handler: httpHandler,
	}

	go func() {
		server.Serve(listener) //nolint
	}()
	defer Stop(logger, server)

	logger.Infof("Moira Api Started (version: %s)", MoiraVersion)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira API shutting down.")
}

// Stop Moira API HTTP server
func Stop(logger moira.Logger, server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Can't stop Moira API correctly: %v", err)
	}
}
