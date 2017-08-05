package master

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker"
	"sync"
	"time"
)

type Worker struct {
	logger   moira.Logger
	database moira.Database
	config   *checker.Config
	lastData int64
	noCache  bool
}

func NewMaster(logger moira.Logger, database moira.Database, config *checker.Config) *Worker {
	return &Worker{
		logger:   logger,
		database: database,
		config:   config,
		lastData: time.Now().UTC().Unix(),
		noCache:  false,
	}
}

func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	var checkerWaitGroup sync.WaitGroup
	checkerWaitGroup.Add(1)
	go worker.noDataChecker(shutdown, &checkerWaitGroup)
	metricEventsChannel := worker.database.SubscribeMetricEvents(shutdown)

	for {
		metricEvent, ok := <-metricEventsChannel
		if !ok {
			worker.logger.Info("Stop checking new events")
			break
		}
		if err := worker.handleMetricEvent(metricEvent); err != nil {
			worker.logger.Errorf("Failed to handle metricEvent", err.Error())
		}
	}
	checkerWaitGroup.Wait()
}

func (worker *Worker) handleMetricEvent(metricEvent *moira.MetricEvent) error {
	worker.lastData = time.Now().UTC().Unix()
	pattern := metricEvent.Pattern
	metric := metricEvent.Metric

	if err := worker.database.AddPatternMetric(pattern, metric); err != nil {
		return err
	}
	triggerIds, err := worker.database.GetPatternTriggerIds(pattern)
	if err != nil {
		return err
	}
	if len(triggerIds) == 0 {
		worker.database.RemovePattern(pattern)
		if err := worker.database.RemovePatternWithMetrics(pattern); err != nil {
			return err
		}
	}

	var waitGroup sync.WaitGroup
	for _, triggerId := range triggerIds {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if worker.noCache {
				if err = worker.database.AddTriggerCheck(triggerId); err != nil {
					worker.logger.Info(err.Error())
				}
			} else {
				//todo triggerId add check cache worker.config.CheckInterval
				if err = worker.database.AddTriggerCheck(triggerId); err != nil {
					worker.logger.Info(err.Error())
				}
			}
		}()
	}
	waitGroup.Wait()
	return nil
}

func (worker *Worker) noDataChecker(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	checkTicker := time.NewTicker(worker.config.NoDataCheckInterval)
	for {
		select {
		case <-shutdown:
			checkTicker.Stop()
			worker.logger.Debugf("Stop Checking nodata")
			return
		case <-checkTicker.C:
			if err := worker.checkNoData(); err != nil {
				worker.logger.Errorf("NoData check failed: %s", err.Error())
			}
		}
	}
}

func (worker *Worker) checkNoData() error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.config.StopCheckingInterval < now {
		worker.logger.Infof("Checking nodata disabled. No metrics for %v seconds", now-worker.lastData)
	} else {
		worker.logger.Info("Checking nodata")
		triggerIds, err := worker.database.GetTriggerIds()
		if err != nil {
			return err
		}
		for _, triggerId := range triggerIds {
			//todo triggerId add check cache 60 seconds
			if err = worker.database.AddTriggerCheck(triggerId); err != nil {
				return err
			}
		}
	}
	return nil
}
