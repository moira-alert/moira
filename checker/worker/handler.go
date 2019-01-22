package worker

import (
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics/graphite"
)

const sleepAfterGetTriggerIDError = time.Millisecond * 500
const sleepWhenNoTriggerToCheck = time.Second * 1
const sleepAfterPanic = time.Second * 1
const sleepAfterCheckingError = time.Second * 5

func (worker *Checker) startTriggerHandler(isRemote bool, metrics *graphite.CheckMetrics) error {
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		default:
			var triggerID string
			var err error
			if isRemote {
				triggerID, err = worker.Database.GetRemoteTriggerToCheck()
			} else {
				triggerID, err = worker.Database.GetTriggerToCheck()
			}
			if err != nil {
				if err == database.ErrNil {
					<-time.After(sleepWhenNoTriggerToCheck)
				} else {
					worker.Logger.Errorf("Failed to handle trigger loop: %s", err.Error())
					<-time.After(sleepAfterGetTriggerIDError)
				}
				continue
			}

			worker.handleTrigger(triggerID, metrics)
		}
	}
}

func (worker *Checker) handleTrigger(triggerID string, metrics *graphite.CheckMetrics) {
	defer func() {
		if r := recover(); r != nil {
			metrics.HandleError.Mark(1)
			worker.Logger.Errorf("Panic while handle trigger %s: message: '%s' stack: %s", triggerID, r, debug.Stack())
			<-time.After(sleepAfterPanic)
		}
	}()
	if err := worker.handleTriggerInLock(triggerID, metrics); err != nil {
		metrics.HandleError.Mark(1)
		worker.Logger.Errorf("Failed to handle trigger: %s error: %s", triggerID, err.Error())
		<-time.After(sleepAfterCheckingError)
	}
}

func (worker *Checker) handleTriggerInLock(triggerID string, metrics *graphite.CheckMetrics) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		defer func() {
			timeSinceStart := time.Since(start)
			metrics.TriggersCheckTime.Update(timeSinceStart)
		}()
		if err := worker.checkTrigger(triggerID); err != nil {
			return err
		}
	}
	return nil
}

func (worker *Checker) checkTrigger(triggerID string) error {
	triggerChecker, err := checker.MakeTriggerChecker(triggerID, worker.Database, worker.Logger, worker.Config, worker.SourceProvider, worker.Metrics)
	if err != nil {
		if err == checker.ErrTriggerNotExists {
			return nil
		}
		return err
	}
	return triggerChecker.Check()
}
