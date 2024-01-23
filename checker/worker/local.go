package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const (
	nodataCheckerLockName = "moira-nodata-checker"
	nodataCheckerLockTTL  = time.Second * 15
	nodataWorkerName      = "NODATA checker"
)

type localChecker struct {
	metrics           *metrics.CheckMetrics
	sourceCheckConfig checker.SourceCheckConfig
	check             *Checker
}

func newLocalChecker(check *Checker, clusterId string) (checkerWorker, error) {
	key := moira.MakeClusterKey(moira.GraphiteLocal, clusterId)

	metrics, err := check.Metrics.GetCheckMetricsBySource(key)
	if err != nil {
		return nil, err
	}

	return &localChecker{
		check:             check,
		sourceCheckConfig: check.Config.SourceCheckConfigs[key],
		metrics:           metrics,
	}, nil
}

func (ch *localChecker) Name() string {
	return "Local"
}

func (ch *localChecker) IsEnabled() bool {
	return true
}

func (ch *localChecker) MaxParallelChecks() int {
	return ch.sourceCheckConfig.MaxParallelChecks
}

func (ch *localChecker) Metrics() *metrics.CheckMetrics {
	return ch.metrics
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
	needToCheckTriggerIDs := check.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		check.Database.AddLocalTriggersToCheck(needToCheckTriggerIDs) //nolint
	}
}
