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
	worker.Logger.Info().Msg("NODATA checker started")
	for {
		select {
		case <-stop:
			worker.Logger.Info().Msg("NODATA checker stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := worker.checkNoData(); err != nil {
				worker.Logger.Error().
					Error(err).
					Msg("NODATA check failed")
			}
		}
	}
}

func (worker *Checker) checkNoData() error {
	now := time.Now().UTC().Unix()
	if worker.lastData+worker.Config.StopCheckingIntervalSeconds < now {
		worker.Logger.Info().
			Int64("no_metrics_for_sec", now-worker.lastData).
			Msg("Checking NODATA disabled. No metrics for some seconds")
	} else {
		worker.Logger.Info().Msg("Checking NODATA")
		triggerIds, err := worker.Database.GetLocalTriggerIDs()
		if err != nil {
			return err
		}
		worker.addTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
