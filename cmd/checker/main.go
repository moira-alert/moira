package main

import (
	"flag"
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker/checker"
	"github.com/moira-alert/moira-alert/checker/master"
	moiraLogging "github.com/moira-alert/moira-alert/logging"
	"github.com/moira-alert/moira-alert/logging/go-logging"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

var (
	configFileName = flag.String("config", "/etc/moira/config.yml", "Path to configuration file")
	printVersion   = flag.Bool("version", false, "Print version and exit")
	verbosityLog   = flag.Bool("-v", false, "Verbosity log")
	triggerId      = flag.String("t", "", "Check single trigger by id and exit")
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

	loggerSettings := config.Checker.getLoggerSettings(verbosityLog)

	logger, err := logging.ConfigureLog(&loggerSettings, "checker")
	if err != nil {
		fmt.Printf("Can not configure log: %s \n", err.Error())
		os.Exit(1)
	}

	if triggerId != nil && *triggerId != "" {
		//todo check single trigger here
		os.Exit(0)
	}

	shutdown := make(chan bool)
	var waitGroup sync.WaitGroup

	masterWorker := master.NewMaster(logger, database)
	checkerWorker := checker.NewChecker(0, logger, database)

	run(masterWorker, shutdown, &waitGroup)
	runCheckers(database, loggerSettings, shutdown, &waitGroup)

	run(checkerWorker, shutdown, &waitGroup)

	logger.Infof("Moira Checker started. Version: %s", Version)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logger.Info(fmt.Sprint(<-ch))
	logger.Infof("Moira Checker shutting down.")
	close(shutdown)
	waitGroup.Wait()
	logger.Infof("Moira Checker stopped. Version: %s", Version)
}

func runCheckers(database moira.Database, loggerSettings moiraLogging.Config, shutdown chan bool, waitGroup *sync.WaitGroup) {
	cpuCount := runtime.NumCPU() - 1
	if cpuCount < 1 {
		cpuCount = 1
	}
	for i := 0; i <= cpuCount; i++ {
		loggerFileName := strings.Split(loggerSettings.LogFile, ".")[0]
		loggerSettings.LogFile = fmt.Sprintf("%s-{%v}", loggerFileName, i)
		logger, err := logging.ConfigureLog(&loggerSettings, fmt.Sprintf("checker-{%v}", i))
		if err != nil {
			fmt.Printf("Can not configure log: %s \n", err.Error())
			os.Exit(1)
		}
		checkerWorker := checker.NewChecker(i, logger, database)
		run(checkerWorker, shutdown, waitGroup)
	}
}

func run(worker moira.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}
