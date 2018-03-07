package worker

import (
	"time"
)

func (worker *Checker) noDataChecker() error {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("NODATA checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkNoData(); err != nil {
				worker.Logger.Errorf("NODATA check failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkNoData() error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.Config.StopCheckingIntervalSeconds < now {
		worker.Logger.Infof("Checking NODATA disabled. No metrics for %v seconds", now-worker.lastData)
	} else {
		worker.Logger.Info("Checking NODATA")
		triggerIds, err := worker.Database.GetTriggerIDs()
		if err != nil {
			return err
		}
		worker.addTriggerIDsIfNeeded(triggerIds, time.Minute)
	}
	return nil
}
