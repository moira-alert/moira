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

func (worker *Checker) startTriggerHandler(isRemote bool) error {
	// TODO: add GetRemoteTriggerToCheck
	for {
		select {
		case <-worker.tomb.Dying():
			return nil
		default:
			triggerID, err := worker.Database.GetTriggerToCheck()
			if err != nil {
				if err == database.ErrNil {
					<-time.After(sleepWhenNoTriggerToCheck)
				} else {
					worker.Logger.Errorf("Failed to handle trigger loop: %s", err.Error())
					<-time.After(sleepAfterGetTriggerIDError)
				}
				continue
			}
			worker.handleTrigger(triggerID, isRemote)
		}
	}
}

func (worker *Checker) handleTrigger(triggerID string, isRemote bool) {
	var errorMetric graphite.Meter
	var triggerType string
	if isRemote {
		errorMetric = worker.Metrics.RemoteHandleError
		triggerType = "remote"
	} else {
		errorMetric = worker.Metrics.HandleError
		triggerType = "local"
	}
	defer func() {
		if r := recover(); r != nil {
			errorMetric.Mark(1)
			worker.Logger.Errorf("Panic while handle %s trigger %s: message: '%s' stack: %s", triggerType, triggerID, r, debug.Stack())
			<-time.After(sleepAfterPanic)
		}
	}()
	if err := worker.handleTriggerInLock(triggerID, isRemote); err != nil {
		worker.Metrics.HandleError.Mark(1)
		worker.Logger.Errorf("Failed to handle trigger: %s error: %s", triggerID, err.Error())
		<-time.After(sleepAfterCheckingError)
	}
}

func (worker *Checker) handleTriggerInLock(triggerID string, isRemote bool) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		defer func() {
			timeSinceStart := time.Since(start)
			if isRemote {
				worker.Metrics.RemoteTriggersCheckTime.Update(timeSinceStart)
				worker.Metrics.RemoteTriggerCheckTime.GetOrAdd(triggerID, triggerID).Update(timeSinceStart)
			} else {
				worker.Metrics.TriggersCheckTime.Update(timeSinceStart)
				worker.Metrics.TriggerCheckTime.GetOrAdd(triggerID, triggerID).Update(timeSinceStart)
			}

		}()
		if err := worker.checkTrigger(triggerID); err != nil {
			return err
		}
	}
	return nil
}

func (worker *Checker) checkTrigger(triggerID string) error {
	defer worker.Database.DeleteTriggerCheckLock(triggerID)
	triggerChecker := checker.TriggerChecker{
		TriggerID:    triggerID,
		Database:     worker.Database,
		Logger:       worker.Logger,
		Config:       worker.Config,
		RemoteConfig: worker.RemoteConfig,
		Metrics:      worker.Metrics,
	}

	err := triggerChecker.InitTriggerChecker()
	if err != nil {
		if err == checker.ErrTriggerNotExists {
			return nil
		}
		return err
	}
	return triggerChecker.Check()
}
