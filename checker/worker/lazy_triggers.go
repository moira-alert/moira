package worker

import (
	"time"

	"github.com/moira-alert/moira/database"
)

const (
	sleepAfterGetUnusedTriggerIDsError = time.Second * 1
	sleepWhenNoUnusedTriggerIDs        = time.Second * 2
	lazyTriggersWorkerTicker           = time.Second * 10
)

func (worker *Checker) lazyTriggersWorker() error {
	checkTicker := time.NewTicker(lazyTriggersWorkerTicker)
	worker.Logger.Infof("Start lazy triggers worker. Check triggers without any subscription every %v", lazyTriggersWorkerTicker)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Lazy triggers worker stopped")
			return nil
		case <-checkTicker.C:
			err := worker.fillLazyTriggerIDs()
			if err == database.ErrNil {
				<-time.After(sleepWhenNoUnusedTriggerIDs)
			} else {
				worker.Logger.Errorf("Failed to get unused triggers: %s", err.Error())
				<-time.After(sleepAfterGetUnusedTriggerIDsError)
			}
		}
	}
}

func (worker *Checker) fillLazyTriggerIDs() error {
	triggerIDs, err := worker.Database.GetUnusedTriggerIDs()
	if err != nil {
		return err
	}
	worker.lazyTriggerIDs = make(map[string]bool)
	for _, triggerID := range triggerIDs {
		worker.lazyTriggerIDs[triggerID] = true
	}
	return nil
}
