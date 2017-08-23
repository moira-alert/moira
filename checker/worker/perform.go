package worker

import (
	"github.com/moira-alert/moira-alert/checker"
	"sync"
	"time"
)

func (worker *Worker) perform(triggerIDs []string, noCache bool, cacheTTL int64, wg *sync.WaitGroup) {
	if noCache {
		for _, id := range triggerIDs {
			wg.Add(1)
			go func(triggerID string) {
				defer wg.Done()
				if err := worker.handleTriggerToCheck(triggerID); err != nil {
					worker.Logger.Errorf("Failed to perform trigger: %s error: %s", triggerID, err.Error())
				}
			}(id)
		}
	} else {
		for _, id := range triggerIDs {
			wg.Add(1)
			go func(triggerID string) {
				defer wg.Done()
				//todo triggerId add check cache cacheTTL seconds
				if err := worker.handleTriggerToCheck(triggerID); err != nil {
					worker.Logger.Errorf("Failed to perform trigger: %s error: %s", triggerID, err.Error())
				}
			}(id)
		}
	}
}

func (worker *Worker) handleTriggerToCheck(triggerId string) error {
	acquired, err := worker.Database.SetTriggerCheckLock(triggerId)
	if err != nil {
		return err
	}
	if acquired {
		start := time.Now()
		if err := worker.checkTrigger(triggerId); err != nil {
			return err
		}
		end := time.Now()
		worker.Metrics.TriggerCheckTime.UpdateSince(start)
		worker.Metrics.TriggerCheckGauge.Update(worker.Metrics.TriggerCheckGauge.Value() + int64(end.Sub(start)))
	}
	return nil
}

func (worker *Worker) checkTrigger(triggerId string) error {
	defer worker.Database.DeleteTriggerCheckLock(triggerId)
	triggerChecker := checker.TriggerChecker{
		TriggerId: triggerId,
		Database:  worker.Database,
		Logger:    worker.Logger,
		Config:    worker.Config,
	}

	err := triggerChecker.InitTriggerChecker()
	if err != nil {
		if err == checker.ErrTriggerNotExists {
			return nil
		}
		return err
	}
	//todo cacheTTL
	return triggerChecker.Check()
}
