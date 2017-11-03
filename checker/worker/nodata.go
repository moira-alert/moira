package worker

import (
	"sync"
	"time"
)

func (worker *Checker) noDataChecker() error {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	var wg sync.WaitGroup
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			wg.Wait()
			worker.Logger.Info("NoData checker stopped")
			return nil
		case <-checkTicker.C:
			if err := worker.checkNoData(&wg); err != nil {
				worker.Logger.Errorf("NoData check failed: %s", err.Error())
			}
		}
	}
}

func (worker *Checker) checkNoData(wg *sync.WaitGroup) error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.Config.StopCheckingInterval < now {
		worker.Logger.Infof("Checking NoData disabled. No metrics for %v seconds", now-worker.lastData)
	} else {
		worker.Logger.Info("Checking NoData")
		triggerIds, err := worker.Database.GetTriggerIDs()
		if err != nil {
			return err
		}
		worker.perform(triggerIds, false, time.Minute, wg)
	}
	return nil
}
