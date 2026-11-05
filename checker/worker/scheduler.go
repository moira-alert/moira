package worker

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics"
	w "github.com/moira-alert/moira/worker"
)

const checkerLockTTL = time.Second * 15

type scheduler struct {
	manager           *WorkerManager
	clusterKey        moira.ClusterKey
	sourceCheckConfig checker.SourceCheckConfig
	name              string
	lockName          string
	validateSource    func() error
	metrics           *metrics.CheckMetrics
}

func newScheduler(manager *WorkerManager, clusterKey moira.ClusterKey, validateSource func() error) (*scheduler, error) {
	metrics, err := manager.Metrics.GetCheckMetricsBySource(clusterKey)
	if err != nil {
		return nil, err
	}

	name := clusterKey.String()
	lockName := "moira-" + name + "-lock"

	return &scheduler{
		manager:           manager,
		sourceCheckConfig: manager.Config.SourceCheckConfigs[clusterKey],
		metrics:           metrics,
		name:              name,
		lockName:          lockName,
		clusterKey:        clusterKey,
		validateSource:    validateSource,
	}, nil
}

func (ch *scheduler) getMaxParallelChecks() int {
	return ch.sourceCheckConfig.MaxParallelChecks
}

func (ch *scheduler) startTriggerScheduler() error {
	w.NewWorker(
		ch.name,
		ch.manager.Logger,
		ch.manager.Database.NewLock(ch.lockName, checkerLockTTL),
		ch.triggerScheduler,
	).Run(ch.manager.tomb.Dying())

	return nil
}

func (ch *scheduler) getTriggersToCheck(count int) ([]string, error) {
	return ch.manager.Database.GetTriggersToCheck(ch.clusterKey, count)
}

func (ch *scheduler) triggerScheduler(stop <-chan struct{}) error {
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

func (ch *scheduler) scheduleTriggersToCheck() error {
	err := ch.validateSource()
	if err != nil {
		ch.manager.Logger.Warning().
			Error(err).
			String("cluster_key", ch.clusterKey.String()).
			Msg("Source is invalid. Stop scheduling trigger checks")

		return nil
	}

	ch.manager.Logger.Debug().
		String("cluster_key", ch.clusterKey.String()).
		Msg("Scheduling triggers")

	triggerIds, err := ch.manager.Database.GetTriggerIDs(ch.clusterKey)
	if err != nil {
		return err
	}

	if ch.manager.Config.SourceSubstitution != nil && ch.clusterKey == ch.manager.Config.SourceSubstitution.ClusterKey {
		return ch.scheduleWithSubstitution(triggerIds)
	}

	err = ch.scheduleTriggerIDsIfNeeded(ch.clusterKey, triggerIds)
	if err != nil {
		return err
	}

	return nil
}

func (ch *scheduler) scheduleWithSubstitution(triggerIDs []string) error {
	substitutionTriggerIDs := make([]string, 0, len(triggerIDs))
	normalTriggerIDs := make([]string, 0, len(triggerIDs))

	for _, triggerID := range triggerIDs {
		if ch.manager.isInSubstitutionSet(triggerID) {
			substitutionTriggerIDs = append(substitutionTriggerIDs, triggerID)
		} else {
			normalTriggerIDs = append(normalTriggerIDs, triggerID)
		}
	}

	substitutionTriggerIDs = ch.manager.filterOutLazyTriggerIDs(substitutionTriggerIDs)
	normalTriggerIDs = ch.manager.filterOutLazyTriggerIDs(normalTriggerIDs)

	targetClusterKey := ch.manager.Config.SourceSubstitution.TargetClusterKey

	if len(substitutionTriggerIDs) > 0 {
		if err := ch.manager.Database.AddTriggersToCheck(targetClusterKey, substitutionTriggerIDs); err != nil {
			return fmt.Errorf("failed to add substitution triggers to check: %w", err)
		}
	}

	if len(normalTriggerIDs) > 0 {
		if err := ch.manager.Database.AddTriggersToCheck(ch.clusterKey, normalTriggerIDs); err != nil {
			return fmt.Errorf("failed to add normal triggers to check: %w", err)
		}
	}

	return nil
}

func (ch *scheduler) scheduleTriggerIDsIfNeeded(clusterKey moira.ClusterKey, triggerIDs []string) error {
	needToCheckTriggerIDs := ch.manager.filterOutLazyTriggerIDs(triggerIDs)
	if len(needToCheckTriggerIDs) > 0 {
		return ch.manager.Database.AddTriggersToCheck(clusterKey, needToCheckTriggerIDs)
	}

	return nil
}
