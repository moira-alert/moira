package worker

import (
	"time"

	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const (
	nodataCheckerLockName = "moira-nodata-checker"
	nodataCheckerLockTTL  = time.Second * 15
	nodataWorkerName      = "NODATA checker"
)

type localChecker struct {
	check *Checker
}

// Returns the new instance of a CheckerWorker that checks graphite local triggers
func NewLocalChecker(check *Checker) CheckerWorker {
	return &localChecker{
		check: check,
	}
}

func (ch *localChecker) Name() string {
	return "Local"
}

func (ch *localChecker) IsEnabled() bool {
	return true
}

func (ch *localChecker) MaxParallelChecks() int {
	return ch.check.Config.MaxParallelLocalChecks
}

func (ch *localChecker) Metrics() *metrics.CheckMetrics {
	return ch.check.Metrics.LocalMetrics
}

// localTriggerGetter starts NODATA checker and manages its subscription in Redis
// to make sure there is always only one working checker
func (ch *localChecker) StartTriggerGetter() error {
	w.NewWorker(
		nodataWorkerName,
		ch.check.Logger,
		ch.check.Database.NewLock(nodataCheckerLockName, nodataCheckerLockTTL),
		ch.localChecker,
	).Run(ch.check.tomb.Dying())

	return nil
}

func (ch *localChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.check.Database.GetLocalTriggersToCheck(count)
}

func (ch *localChecker) localChecker(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.check.Config.NoDataCheckInterval)
	ch.check.Logger.Info().Msg("Local checker started")
	for {
		select {
		case <-stop:
			ch.check.Logger.Info().Msg("Local checker stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := ch.addLocalTriggersToCheckQueue(); err != nil {
				ch.check.Logger.Error().
					Error(err).
					Msg("Local check failed")
			}
		}
	}
}

func (ch *localChecker) addLocalTriggersToCheckQueue() error {
	now := time.Now().UTC().Unix()
	if ch.check.lastData+ch.check.Config.StopCheckingIntervalSeconds < now {
		ch.check.Logger.Info().
			Int64("no_metrics_for_sec", now-ch.check.lastData).
			Msg("Checking Local disabled. No metrics for some seconds")
		return nil
	}

	ch.check.Logger.Info().Msg("Checking Local")
	triggerIds, err := ch.check.Database.GetLocalTriggerIDs()
	if err != nil {
		return err
	}
	ch.check.addLocalTriggerIDsIfNeeded(triggerIds)

	return nil
}

func (check *Checker) addLocalTriggerIDsIfNeeded(triggerIDs []string) {
	needToCheckTriggerIDs := check.getTriggerIDsToCheck(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		check.Database.AddLocalTriggersToCheck(needToCheckTriggerIDs) //nolint
	}
}
