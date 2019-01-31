package worker

import (
	. "github.com/moira-alert/moira/worker"
	"time"
)

const nodataCheckerLockName = "moira-nodata-checker"
const nodataCheckerLockTTL = time.Second * 15

func (worker *Checker) noDataChecker(stop <-chan struct{}) {
	checkTicker := time.NewTicker(worker.Config.NoDataCheckInterval)
	defer checkTicker.Stop()
	worker.Logger.Info("NODATA checker started")
	for {
		select {
		case <-stop:
			worker.Logger.Info("NODATA checker stopped")
			return
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

// runNodataChecker starts NODATA checker and manages its subscription in Redis
// to make sure there is always only one working checker
func (worker *Checker) runNodataChecker() error {
	NewWorker(
		"NOData checker",
		worker.Logger,
		worker.Database.NewLock(nodataCheckerLockName, nodataCheckerLockTTL),
		worker.noDataChecker,
	).Run(worker.tomb.Dying())

	return nil
}
