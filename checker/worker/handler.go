package worker

import (
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database"
)

const sleepAfterGetTriggerIDError = time.Millisecond * 500
const sleepWhenNoTriggerToCheck = time.Second * 1
const sleepAfterPanic = time.Second * 1
const sleepAfterCheckingError = time.Second * 5

func (worker *Checker) startTriggerHandler(isRemote bool) error {
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
					if isRemote {
						worker.Logger.Errorf("Failed to handle remote trigger loop: %s", err.Error())
					} else {
						worker.Logger.Errorf("Failed to handle local trigger loop: %s", err.Error())
					}
					<-time.After(sleepAfterGetTriggerIDError)
				}
				continue
			}
			if isRemote {
				worker.handleRemoteTrigger(triggerID)
			} else {
				worker.handleTrigger(triggerID)
			}
		}
	}
}

func (worker *Checker) handleTrigger(triggerID string) {
	defer func() {
		if r := recover(); r != nil {
			worker.Metrics.HandleError.Mark(1)
			worker.Logger.Errorf("Panic while handle local trigger %s: message: '%s' stack: %s", triggerID, r, debug.Stack())
			<-time.After(sleepAfterPanic)
		}
	}()
	if err := worker.handleTriggerInLock(triggerID); err != nil {
		worker.Metrics.HandleError.Mark(1)
		worker.Logger.Errorf("Failed to handle local trigger: %s error: %s", triggerID, err.Error())
		<-time.After(sleepAfterCheckingError)
	}
}

func (worker *Checker) handleRemoteTrigger(triggerID string) {
	defer func() {
		if r := recover(); r != nil {
			worker.Metrics.RemoteHandleError.Mark(1)
			worker.Logger.Errorf("Panic while handle remote trigger %s: message: '%s' stack: %s", triggerID, r, debug.Stack())
			<-time.After(sleepAfterPanic)
		}
	}()
	if err := worker.handleRemoteTriggerInLock(triggerID); err != nil {
		worker.Metrics.RemoteHandleError.Mark(1)
		worker.Logger.Errorf("Failed to handle remote trigger: %s error: %s", triggerID, err.Error())
		<-time.After(sleepAfterCheckingError)
	}
}

func (worker *Checker) handleTriggerInLock(triggerID string) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		defer func() {
			timeSinceStart := time.Since(start)
			worker.Metrics.TriggersCheckTime.Update(timeSinceStart)
			worker.Metrics.TriggerCheckTime.GetOrAdd(triggerID, triggerID).Update(timeSinceStart)
		}()
		if err := worker.checkTrigger(triggerID); err != nil {
			return err
		}
	}
	return nil
}

func (worker *Checker) handleRemoteTriggerInLock(triggerID string) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		defer func() {
			timeSinceStart := time.Since(start)
			worker.Metrics.RemoteTriggersCheckTime.Update(timeSinceStart)
			worker.Metrics.RemoteTriggerCheckTime.GetOrAdd(triggerID, triggerID).Update(timeSinceStart)
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
