package worker

import (
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
)

func (worker *Checker) startTriggerHandler(isRemote bool) error {
	var ch <-chan string
	if isRemote {
		ch = worker.remoteTriggersToCheck
	} else {
		ch = worker.triggersToCheck
	}
	for triggerID := range ch {
		worker.handleTrigger(triggerID, isRemote)
	}
	return nil
}

func (worker *Checker) handleTrigger(triggerID string, isRemote bool) {
	var errorMetric graphite.Meter
	triggerType := " "
	if isRemote {
		errorMetric = worker.Metrics.RemoteHandleError
		triggerType = " remote"
	} else {
		errorMetric = worker.Metrics.HandleError
	}
	defer func() {
		if r := recover(); r != nil {
			errorMetric.Mark(1)
			worker.Logger.Errorf("Panic while handle%s trigger %s: message: '%s' stack: %s", triggerType, triggerID, r, debug.Stack())
		}
	}()
	if err := worker.handleTriggerInLock(triggerID, isRemote); err != nil {
		errorMetric.Mark(1)
		worker.Logger.Errorf("Failed to handle%s trigger: %s error: %s", triggerType, triggerID, err.Error())
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
