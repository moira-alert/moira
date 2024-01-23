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
	check             *Checker
	name              string
	lockName          string
	clusterKey        moira.ClusterKey
	validateSource    func() error
}

func newUniversalChecker(check *Checker, clusterKey moira.ClusterKey, validateSource func() error) (checkerWorker, error) {
	metrics, err := check.Metrics.GetCheckMetricsBySource(clusterKey)
	if err != nil {
		return nil, err
	}

	name := clusterKey.TriggerSource.String()
	if clusterKey.ClusterId != "default" {
		name = name + ":" + clusterKey.ClusterId
	}

	lockName := "moira-" + name + "-lock"

	return &universalChecker{
		check:             check,
		sourceCheckConfig: check.Config.SourceCheckConfigs[clusterKey],
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

func (ch *universalChecker) IsEnabled() bool {
	return ch.sourceCheckConfig.Enabled
}

func (ch *universalChecker) MaxParallelChecks() int {
	return ch.sourceCheckConfig.MaxParallelChecks
}

func (ch *universalChecker) Metrics() *metrics.CheckMetrics {
	return ch.metrics
}

func (ch *universalChecker) StartTriggerGetter() error {
	w.NewWorker(
		ch.name,
		ch.check.Logger,
		ch.check.Database.NewLock(ch.lockName, checkerLockTTL),
		ch.triggerScheduler,
	).Run(ch.check.tomb.Dying())

	return nil
}

func (ch *universalChecker) GetTriggersToCheck(count int) ([]string, error) {
	return ch.check.Database.GetTriggersToCheck(ch.clusterKey, count)
}

func (ch *universalChecker) triggerScheduler(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(ch.sourceCheckConfig.CheckInterval)

	ch.check.Logger.Info().Msg(ch.name + " started")
	for {
		select {
		case <-stop:
			ch.check.Logger.Info().Msg(ch.name + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := ch.scheduleTriggersToCheck(); err != nil {
				ch.check.Logger.Error().
					Error(err).
					Msg(ch.name + " trigger failed")
			}
		}
	}
}

func (ch *universalChecker) scheduleTriggersToCheck() error {
	err := ch.validateSource()
	if err != nil {
		ch.check.Logger.Info().
			Error(err).
			String("cluster_key", ch.clusterKey.String()).
			Msg("Source is invalid. Stop scheduling trigger checks")
		return nil
	}

	ch.check.Logger.Debug().
		String("cluster_key", ch.clusterKey.String()).
		Msg("Scheduling triggers")

	triggerIds, err := ch.check.Database.GetRemoteTriggerIDs()
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
	needToCheckTriggerIDs := ch.check.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		return ch.check.Database.AddTriggersToCheck(clusterKey, needToCheckTriggerIDs)
	}
	return nil
}
