package worker

import (
	"github.com/moira-alert/moira/database"
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
	lock := worker.Database.NewLock(nodataCheckerLockName, nodataCheckerLockTTL)
	for {
		lost, err := lock.Acquire(worker.tomb.Dying())
		if err != nil {
			if err == database.ErrLockAcquireInterrupted {
				return nil
			}

			worker.Logger.Warningf("Could not acquire lock for NODATA checker, err %s", err)
			continue
		}

		stop := make(chan struct{})
		go worker.noDataChecker(stop)
		select {
		case <-worker.tomb.Dying():
			close(stop)
			lock.Release()
			return nil
		case <-lost:
			close(stop)
			lock.Release()
			continue
		}
	}
}
