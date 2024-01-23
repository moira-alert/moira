package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const checkerLockTTL = time.Second * 15

type universalChecker struct {
	metrics           *metrics.CheckMetrics
	sourceCheckConfig checker.SourceCheckConfig
	manager             *WorkerManager
	name              string
	lockName          string
	clusterKey        moira.ClusterKey
	validateSource    func() error
}

func newUniversalChecker(manager *WorkerManager, clusterKey moira.ClusterKey, validateSource func() error) (*universalChecker, error) {
	metrics, err := manager.Metrics.GetCheckMetricsBySource(clusterKey)
	if err != nil {
		return nil, err
	}

	name := clusterKey.TriggerSource.String()
	if clusterKey.ClusterId != "default" {
		name = name + ":" + clusterKey.ClusterId
	}

	lockName := "moira-" + name + "-lock"

	return &universalChecker{
		manager:             manager,
		sourceCheckConfig: manager.Config.SourceCheckConfigs[clusterKey],
		metrics:           metrics,
		name:              name,
		lockName:          lockName,
		clusterKey:        clusterKey,
		validateSource:    validateSource,
	}, nil
}

func (ch *universalChecker) Name() string {
	return ch.name
}

func (ch *universalChecker) MaxParallelChecks() int {
	return ch.sourceCheckConfig.MaxParallelChecks
}

func (ch *universalChecker) Metrics() *metrics.CheckMetrics {
	return ch.metrics
}

func (ch *universalChecker) StartTriggerScheduler() error {
	w.NewWorker(
		ch.name,
		ch.manager.Logger,
		ch.manager.Database.NewLock(ch.lockName, checkerLockTTL),
		ch.triggerScheduler,
	).Run(ch.manager.tomb.Dying())

	return nil
}

func (ch *universalChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.manager.Database.GetTriggersToCheck(ch.clusterKey, count)
}

func (ch *universalChecker) triggerScheduler(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.sourceCheckConfig.CheckInterval)

	ch.manager.Logger.Info().Msg(ch.name + " started")
	for {
		select {
		case <-stop:
			ch.manager.Logger.Info().Msg(ch.name + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := ch.scheduleTriggersToCheck(); err != nil {
				ch.manager.Logger.Error().
					Error(err).
					Msg(ch.name + " trigger failed")
			}
		}
	}
}

func (ch *universalChecker) scheduleTriggersToCheck() error {
	err := ch.validateSource()
	if err != nil {
		ch.manager.Logger.Info().
			Error(err).
			String("cluster_key", ch.clusterKey.String()).
			Msg("Source is invalid. Stop scheduling trigger checks")
		return nil
	}

	ch.manager.Logger.Debug().
		String("cluster_key", ch.clusterKey.String()).
		Msg("Scheduling triggers")

	triggerIds, err := ch.manager.Database.GetRemoteTriggerIDs()
	if err != nil {
		return err
	}

	err = ch.addTriggerIDsIfNeeded(ch.clusterKey, triggerIds)
	if err != nil {
		return err
	}

	return nil
}

func (ch *universalChecker) addTriggerIDsIfNeeded(clusterKey moira.ClusterKey, triggerIDs []string) error {
	needToCheckTriggerIDs := ch.manager.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		return ch.manager.Database.AddTriggersToCheck(clusterKey, needToCheckTriggerIDs)
	}
	return nil
}
