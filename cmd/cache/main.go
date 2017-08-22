package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/cache/connection"
	"github.com/moira-alert/moira-alert/cache/heartbeat"
	"github.com/moira-alert/moira-alert/cache/matched_metrics"
	"github.com/moira-alert/moira-alert/cache/patterns"
	"github.com/moira-alert/moira-alert/cmd"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/metrics/graphite/atomic"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
)

var (
	logger         moira.Logger
	database       moira.Database
	cacheMetrics   *graphite.CacheMetrics
	cacheStorage   *cache.Storage
	patternStorage *cache.PatternStorage

	shutdown  chan bool
	waitGroup sync.WaitGroup

	configFileName         = flag.String("config", "/etc/moira/config.yml", "path config file")
	printVersion           = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag = flag.Bool("default-config", false, "Print default config and exit")

	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira Cache")
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

	loggerSettings := config.Logger.GetSettings(false)

	logger, err = logging.ConfigureLog(&loggerSettings, "cache")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not configure log: %s\n", err.Error())
		os.Exit(1)
	}

	cacheMetrics = metrics.ConfigureCacheMetrics()
	databaseMetrics := metrics.ConfigureDatabaseMetrics()
	metrics.Init(config.Graphite.GetSettings(), logger, "cache")

	database = redis.NewDatabase(logger, config.Redis.GetSettings(), databaseMetrics)

	retentionConfigFile, err := os.Open(config.Cache.RetentionConfig)
	if err != nil {
		logger.Fatalf("Error open retentions file [%s]: %s", config.Cache.RetentionConfig, err.Error())
	}

	cacheStorage, err = cache.NewCacheStorage(cacheMetrics, retentionConfigFile)
	if err != nil {
		logger.Fatalf("Failed to initialize cache with config [%s]: %s", config.Cache.RetentionConfig, err.Error())
	}

	patternStorage, err = cache.NewPatternStorage(database, cacheMetrics, logger)
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}

	shutdown = make(chan bool)

	refreshPatternWorker := patterns.NewRefreshPatternWorker(database, cacheMetrics, logger, patternStorage)
	heartbeatWorker := heartbeat.NewHeartbeatWorker(database, cacheMetrics, logger)
	atomicMetricsWorker := atomic.NewAtomicMetricsWorker(cacheMetrics)

	run(refreshPatternWorker, shutdown, &waitGroup)
	run(heartbeatWorker, shutdown, &waitGroup)
	run(atomicMetricsWorker, shutdown, &waitGroup)

	listener, err := connection.NewListener(config.Cache.Listen, logger, patternStorage)
	if err != nil {
		logger.Fatalf("Failed to start listen: %s", err.Error())
	}

	metricsChan := make(chan *moira.MatchedMetric, 10)
	matchedMetricsProcessor := matchedmetrics.NewMatchedMetricsProcessor(cacheMetrics, logger, database, cacheStorage)

	waitGroup.Add(1)
	go matchedMetricsProcessor.Run(metricsChan, &waitGroup)

	waitGroup.Add(1)
	go listener.Listen(metricsChan, &waitGroup, shutdown)

	logger.Infof("Moira Cache started. Version: %s", MoiraVersion)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Cache shutting down.")
	close(shutdown)
	waitGroup.Wait()
	logger.Infof("Moira Cache stopped. Version: %s", MoiraVersion)
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
