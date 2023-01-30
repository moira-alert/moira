package worker

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics"
)

const sleepAfterCheckingError = time.Second * 2

// startTriggerHandler is blocking func
func (worker *Checker) startTriggerHandler(triggerIDsToCheck <-chan string, metrics *metrics.CheckMetrics) error {
	for {
		triggerID, ok := <-triggerIDsToCheck
		if !ok {
			return nil
		}
		err := worker.handleTrigger(triggerID, metrics)
		if err != nil {
			metrics.HandleError.Mark(1)

			worker.Logger.Error().
				String(moira.LogFieldNameTriggerID, triggerID).
				Error(err).
				Msg("Failed to handle trigger")

			<-time.After(sleepAfterCheckingError)
		}
	}
}

func (worker *Checker) handleTrigger(triggerID string, metrics *metrics.CheckMetrics) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: '%s' stack: %s", r, debug.Stack())
		}
	}()
	err = worker.handleTriggerInLock(triggerID, metrics)
	return err
}

func (worker *Checker) handleTriggerInLock(triggerID string, metrics *metrics.CheckMetrics) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerID)
	if err != nil {
		return err
	}
	if acquired {
		startedAt := time.Now()
		defer metrics.TriggersCheckTime.UpdateSince(startedAt)
		if err := worker.checkTrigger(triggerID); err != nil {
			return err
		}
	}
	return nil
}

func (worker *Checker) checkTrigger(triggerID string) error {
	defer worker.Database.DeleteTriggerCheckLock(triggerID) //nolint
	triggerChecker, err := checker.MakeTriggerChecker(triggerID, worker.Database, worker.Logger, worker.Config, worker.SourceProvider, worker.Metrics)
	if err != nil {
		if err == checker.ErrTriggerNotExists {
			return nil
		}
		return err
	}
	return triggerChecker.Check()
}
