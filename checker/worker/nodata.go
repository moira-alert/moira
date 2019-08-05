package worker

import (
	"time"

	w "github.com/moira-alert/moira/worker"
)

const (
	nodataCheckerLockName = "moira-nodata-checker"
	nodataCheckerLockTTL  = time.Second * 15
	nodataWorkerName      = "NODATA checker"
)

// localTriggerGetter starts NODATA checker and manages its subscription in Redis
// to make sure there is always only one working checker
func (worker *Checker) localTriggerGetter() error {
	w.NewWorker(
		nodataWorkerName,
		worker.Logger,
		worker.Database.NewLock(nodataCheckerLockName, nodataCheckerLockTTL),
		worker.noDataChecker,
	).Run(worker.tomb.Dying())

	return nil
}

func (worker *Checker) noDataChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	defer checkTicker.Stop()
	worker.Logger.Info("NODATA checker started")
	for {
		select {
		case <-stop:
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
		triggerIds, err := worker.Database.GetLocalTriggerIDs()
		if err != nil {
			return err
		}
		worker.addTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
