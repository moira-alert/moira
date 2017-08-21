package worker

import (
	"gopkg.in/tomb.v2"
	"time"
)

func (worker *Worker) noDataChecker() error {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	var tomb2 tomb.Tomb
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			tomb2.Wait()
			worker.Logger.Debugf("NoData checker stopped")
			return nil
		case <-checkTicker.C:
			if !tomb2.Alive() {
				tomb2 = tomb.Tomb{}
			}
			if err := worker.checkNoData(&tomb2); err != nil {
				worker.Logger.Errorf("NoData check failed: %s", err.Error())
			}
		}
	}
	return nil
}

func (worker *Worker) checkNoData(tomb *tomb.Tomb) error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.Config.StopCheckingInterval < now {
		worker.Logger.Infof("Checking NoData disabled. No metrics for %v seconds", now-worker.lastData)
	} else {
		worker.Logger.Info("Checking NoData")
		triggerIds, err := worker.Database.GetTriggerIds()
		if err != nil {
			return err
		}
		worker.perform(triggerIds, false, 60, tomb)
	}
	return nil
}
