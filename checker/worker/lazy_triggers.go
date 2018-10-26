package worker

import (
	"time"
)

const lazyTriggersWorkerTicker = time.Second * 10

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
			if err != nil {
				worker.Logger.Errorf("Failed to get unused triggers: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) fillLazyTriggerIDs() error {
	triggerIDs, err := worker.Database.GetUnusedTriggerIDs()
	if err != nil {
		return err
	}
	newLazyTriggerIDs := make(map[string]bool)
	for _, triggerID := range triggerIDs {
		newLazyTriggerIDs[triggerID] = true
	}
	worker.lazyTriggerIDs = newLazyTriggerIDs
	worker.Metrics.UnusedTriggersCount.Update(int64(len(worker.lazyTriggerIDs)))
	return nil
}
