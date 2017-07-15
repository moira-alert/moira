package main

import (
	"flag"
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"github.com/moira-alert/moira-alert/cache/connection"
	"github.com/moira-alert/moira-alert/cache/heartbeat"
	"github.com/moira-alert/moira-alert/cache/matched_metrics"
	"github.com/moira-alert/moira-alert/cache/patterns"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/metrics/graphite/atomic"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/rcrowley/goagain"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
)

var (
	logger         moira.Logger
	database       moira.Database
	metric         *graphite.CacheMetrics
	cacheStorage   *cache.CacheStorage
	patternStorage *cache.PatternStorage
	listener       net.Listener

	shutdown  chan bool
	waitGroup sync.WaitGroup
)

var (
	configFileName = flag.String("config", "/etc/moira/config.yml", "path config file")
	logParseErrors = flag.Bool("logParseErrors", false, "enable logging metric parse errors")
	printVersion   = flag.Bool("version", false, "Print version and exit")
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

	logger, err = configureLog(&config.Cache)
	if err != nil {
		fmt.Printf("Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	err = ioutil.WriteFile(config.Cache.PidFile, []byte(fmt.Sprint(syscall.Getpid())), 0644)
	if err != nil {
		logger.Fatalf("Error writing pid file [%s]: %s", config.Cache.PidFile, err.Error())
	}

	metric = metrics.ConfigureCacheMetrics()
	metrics.Init(config.Graphite.getSettings(), logger)

	database = redis.Init(logger, config.Redis.getSettings(), &graphite.NotifierMetrics{}) //todo костыль

	cacheStorage, err = cache.NewCacheStorage(database, metric, config.Cache.RetentionConfig)
	if err != nil {
		logger.Fatalf("Failed to initialize cache with config [%s]: %s", config.Cache.RetentionConfig, err.Error())
	}

	patternStorage, err = cache.NewPatternStorage(database, metric, logger, *logParseErrors)
	if err != nil {
		logger.Fatalf("Failed to refresh pattern storage: %s", err.Error())
	}

	shutdown = make(chan bool)

	refreshPatternWorker := patterns.NewRefreshPatternWorker(database, metric, logger, patternStorage)
	heartbeatWorker := heartbeat.NewHeartbeatWorker(database, metrics, logger)
	atomicMetricsWorker := atomic.NewAtomicMetricsWorker(metric)

	run(refreshPatternWorker, shutdown, &waitGroup)
	run(heartbeatWorker, shutdown, &waitGroup)
	run(atomicMetricsWorker, shutdown, &waitGroup)

	listener, err = createListener(config)
	if err != nil {
		logger.Fatalf("Failed to start listen: %s", err.Error())
	}

	waitGroup.Add(1)
	go serve()

	if _, err := goagain.Wait(listener); err != nil {
		log.Fatalf("failed to block main goroutine: %s", err.Error())
	}

	log.Printf("shutting down")
	if err := listener.Close(); err != nil {
		log.Fatalf("failed to stop listening: %s", err.Error())
	}
	waitGroup.Wait()
	log.Printf("shutdown complete")
}

func serve() {
	defer waitGroup.Done()
	metricsChan := make(chan *moira.MatchedMetric, 10)
	matchedMetricsProcessor := matchedmetrics.NewMatchedMetricsProcessor(metric, logger, database, cacheStorage)

	waitGroup.Add(1)
	go matchedMetricsProcessor.Run(metricsChan, &waitGroup)

	handleConnections(metricsChan)
	close(metricsChan)
}

func createListener(config *config) (net.Listener, error) {
	listen := config.Cache.Listen
	listener, err := goagain.Listener()
	if err != nil {
		listener, err = net.Listen("tcp", listen)
		if err != nil {
			return nil, fmt.Errorf("Failed to listen on [%s]: %s", listen, err.Error())
		}
		logger.Infof("listening on %s", listen)
	} else {
		logger.Infof("resuming listening on %s", listen)
		if err := goagain.Kill(); err != nil {
			return nil, fmt.Errorf("Failed to kill parent process: %s", err.Error())
		}
	}
	return listener, nil
}

func handleConnections(metricsChan chan *moira.MatchedMetric) {
	connectionHandler := connection.NewConnectionHandler(logger, patternStorage)
	var handlerWG sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			if goagain.IsErrClosing(err) {
				log.Println("Listener closed")
				close(shutdown)
				break
			}
			log.Printf("failed to accept connection: %s", err.Error())
			continue
		}
		handlerWG.Add(1)
		go connectionHandler.HandleConnection(conn, metricsChan, shutdown, &handlerWG)
	}
	handlerWG.Wait()
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
