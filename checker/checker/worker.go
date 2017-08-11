package checker_worker

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"math/rand"
	"sync"
	"time"
)

var errorTimeout = time.Second * 10
var performTimeout = time.Millisecond * 10

type Worker struct {
	checkerNumber int
	logger        moira.Logger
	database      moira.Database
	metrics       *graphite.CheckerMetrics
	config        *checker.Config
}

func NewChecker(checkerNumber int, logger moira.Logger, database moira.Database, metrics *graphite.CheckerMetrics, config *checker.Config) *Worker {
	return &Worker{
		checkerNumber: checkerNumber,
		logger:        logger,
		database:      database,
		metrics:       metrics,
		config:        config,
	}
}

func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	rand.Seed(time.Now().UTC().UnixNano())
	for {
		select {
		case <-shutdown:
			worker.logger.Info("Stop checker service")
			return
		default:
			hasNewTrigger, err := worker.perform()
			if err != nil {
				worker.logger.Errorf("Failed to perform triggers check: %s", err.Error())
				worker.metrics.CheckerError.Mark(1)
				<-time.After(errorTimeout)
				continue
			}
			if !hasNewTrigger {
				durationAfterCoefficient := rand.Intn(10) + 10
				<-time.After(performTimeout * time.Duration(durationAfterCoefficient))
				continue
			}
		}
	}
}

func (worker *Worker) perform() (bool, error) {
	triggerId, err := worker.database.GetTriggerToCheck()
	if err != nil || triggerId == nil {
		return false, err
	}
	err = worker.handleTriggerToCheck(*triggerId)
	return true, err
}

func (worker *Worker) handleTriggerToCheck(triggerId string) error {
	acquired, err := worker.database.SetTriggerCheckLock(triggerId)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		if err := worker.checkTrigger(triggerId); err != nil {
			return err
		}
		end := time.Now()
		worker.metrics.TriggerCheckTime.UpdateSince(start)
		worker.metrics.TriggerCheckGauge.Update(worker.metrics.TriggerCheckGauge.Value() + int64(start.Sub(end)))
	}
	return nil
}

func (worker *Worker) checkTrigger(triggerId string) error {
	defer worker.database.DeleteTriggerCheckLock(triggerId)
	triggerChecker := checker.TriggerChecker{
		TriggerId: triggerId,
		Database:  worker.database,
		Logger:    worker.logger,
		Config:    worker.config,
	}
	//todo cacheTTL
	return triggerChecker.Check(nil, nil)
}
