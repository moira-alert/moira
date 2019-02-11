package worker

import (
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira/checker"
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
			triggerIDs, err := worker.getTriggersToCheck(isRemote)
			if err != nil {
				worker.Logger.Errorf("Failed to handle trigger loop: %s", err.Error())
				<-time.After(sleepAfterGetTriggerIDError)
				continue
			}
			if len(triggerIDs) == 0 {
				<-time.After(sleepWhenNoTriggerToCheck)
				continue
			}
			worker.handleTrigger(triggerIDs[0], metrics)
		}
	}
}

func (worker *Checker) getTriggersToCheck(isRemote bool) ([]string, error) {
	if isRemote {
		return worker.Database.GetRemoteTriggersToCheck(1)
	}
	return worker.Database.GetTriggersToCheck(1)
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
	defer worker.Database.DeleteTriggerCheckLock(triggerID)
	triggerChecker, err := checker.MakeTriggerChecker(triggerID, worker.Database, worker.Logger, worker.Config, worker.SourceProvider, worker.Metrics)
	if err != nil {
		if err == checker.ErrTriggerNotExists {
			return nil
		}
		return err
	}
	return triggerChecker.Check()
}
