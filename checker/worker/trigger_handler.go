package worker

import (
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics"
)

const sleepAfterCheckingError = time.Second * 2

// startTriggerHandler is a blocking func
func (manager *WorkerManager) startTriggerHandler(triggerIDsToCheck <-chan string, metrics *metrics.CheckMetrics) error {
	for {
		triggerID, ok := <-triggerIDsToCheck
		if !ok {
			return nil
		}

		err := manager.handleTrigger(triggerID, metrics)
		if err != nil {
			metrics.HandleError.Mark(1)

			manager.Logger.Error().
				String(moira.LogFieldNameTriggerID, triggerID).
				Error(err).
				Msg("Failed to handle trigger")

			<-time.After(sleepAfterCheckingError)
		}
	}
}

func (manager *WorkerManager) handleTrigger(triggerID string, metrics *metrics.CheckMetrics) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: '%s' stack: %s", r, debug.Stack())
		}
	}()
	err = manager.handleTriggerInLock(triggerID, metrics)
	return err
}

func (manager *WorkerManager) handleTriggerInLock(triggerID string, metrics *metrics.CheckMetrics) error {
	acquired, err := manager.Database.SetTriggerCheckLock(triggerID)
	defer manager.Database.DeleteTriggerCheckLock(triggerID) //nolint

	if err != nil {
		return err
	}

	if !acquired {
		return nil
	}

	startedAt := time.Now()
	defer metrics.TriggersCheckTime.UpdateSince(startedAt)

	err = manager.checkTrigger(triggerID)
	return err
}

func (manager *WorkerManager) checkTrigger(triggerID string) error {
	triggerChecker, err := checker.MakeTriggerChecker(
		triggerID,
		manager.Database,
		manager.Logger,
		manager.Config,
		manager.SourceProvider,
		manager.Metrics,
	)

	if errors.Is(err, checker.ErrTriggerNotExists) {
		return nil
	}
	if err != nil {
		return err
	}
	return triggerChecker.Check()
}
