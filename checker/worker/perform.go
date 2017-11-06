package worker

import (
	"github.com/moira-alert/moira/checker"
	"runtime/debug"
	"sync"
	"time"
)

func (worker *Checker) perform(triggerIDs []string, noCache bool, cacheTTL time.Duration, wg *sync.WaitGroup) {
	if noCache {
		for _, triggerID := range triggerIDs {
			wg.Add(1)
			go worker.handle(triggerID, wg)
		}
	} else {
		for _, triggerID := range triggerIDs {
			wg.Add(1)
			if worker.needHandleTrigger(triggerID, cacheTTL) {
				go worker.handle(triggerID, wg)
			}
		}
	}
}

func (worker *Checker) needHandleTrigger(triggerID string, cacheTTL time.Duration) bool {
	err := worker.Cache.Add(triggerID, true, cacheTTL)
	return err == nil
}

func (worker *Checker) handle(triggerID string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			worker.Metrics.HandleError.Mark(1)
			worker.Logger.Errorf("Panic while perform trigger %s: message: '%s' stack: %s", triggerID, r, debug.Stack())
		}
	}()
	if err := worker.handleTriggerToCheck(triggerID); err != nil {
		worker.Metrics.HandleError.Mark(1)
		worker.Logger.Errorf("Failed to perform trigger: %s error: %s", triggerID, err.Error())
	}
}

func (worker *Checker) handleTriggerToCheck(triggerID string) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		defer worker.Metrics.TriggerCheckTime.UpdateSince(start)
		if err := worker.checkTrigger(triggerID); err != nil {
			return err
		}
	}
	return nil
}

func (worker *Checker) checkTrigger(triggerID string) error {
	defer worker.Database.DeleteTriggerCheckLock(triggerID)
	triggerChecker := checker.TriggerChecker{
		TriggerID: triggerID,
		Database:  worker.Database,
		Logger:    worker.Logger,
		Config:    worker.Config,
		Metrics:   worker.Metrics,
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
