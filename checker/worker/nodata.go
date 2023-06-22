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
func (check *Checker) localTriggerGetter() error {
	w.NewWorker(
		nodataWorkerName,
		check.Logger,
		check.Database.NewLock(nodataCheckerLockName, nodataCheckerLockTTL),
		check.noDataChecker,
	).Run(check.tomb.Dying())

	return nil
}

func (check *Checker) noDataChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(check.Config.NoDataCheckInterval)
	check.Logger.Info().Msg("NODATA checker started")
	for {
		select {
		case <-stop:
			check.Logger.Info().Msg("NODATA checker stopped")
			checkTicker.Stop()
			return nil
		case <-checkTicker.C:
			if err := check.checkNoData(); err != nil {
				check.Logger.Error().
					Error(err).
					Msg("NODATA check failed")
			}
		}
	}
}

func (check *Checker) checkNoData() error {
	now := time.Now().UTC().Unix()
	if check.lastData+check.Config.StopCheckingIntervalSeconds < now {
		check.Logger.Info().
			Int64("no_metrics_for_sec", now-check.lastData).
			Msg("Checking NODATA disabled. No metrics for some seconds")
	} else {
		check.Logger.Info().Msg("Checking NODATA")
		triggerIds, err := check.Database.GetLocalTriggerIDs()
		if err != nil {
			return err
		}
		check.addTriggerIDsIfNeeded(triggerIds)
	}
	return nil
}
