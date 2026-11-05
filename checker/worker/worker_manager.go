package worker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moira-alert/moira/metrics"

	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
)

// WorkerManager represents workers for periodically triggers checking based by new events.
type WorkerManager struct {
	Logger   moira.Logger
	Database moira.Database

	Config         *checker.Config
	SourceProvider *metricSource.SourceProvider
	Metrics        *metrics.CheckerMetrics

	TriggerCache      *cache.Cache
	LazyTriggersCache *cache.Cache
	PatternCache      *cache.Cache
	lazyTriggerIDs    atomic.Value
	lastData          int64
	substitutionSet   map[string]struct{}
	tomb              tomb.Tomb
	substitutionMutex sync.RWMutex
}

// StartWorkers start schedule new MetricEvents and check for NODATA triggers.
func (manager *WorkerManager) StartWorkers() error {
	var err error

	err = manager.startLazyTriggers()
	if err != nil {
		return err
	}

	err = manager.startLocalMetricEvents()
	if err != nil {
		return err
	}

	if manager.Config.SourceSubstitution != nil {
		if err := manager.initSubstitutionSet(); err != nil {
			return fmt.Errorf("failed to initialize substitution set: %w", err)
		}
	}

	for clusterKey := range manager.Config.SourceCheckConfigs {
		validator, err := manager.makeSourceValidator(clusterKey)
		if err != nil {
			return err
		}

		checker, err := newScheduler(manager, clusterKey, validator)
		if err != nil {
			return err
		}

		err = manager.startCheckerWorker(checker)
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *WorkerManager) initSubstitutionSet() error {
	cfg := manager.Config.SourceSubstitution
	targetClusterKey := cfg.TargetClusterKey

	sourceExists := false
	for ck := range manager.Config.SourceCheckConfigs {
		if ck == targetClusterKey {
			sourceExists = true
			break
		}
	}
	if !sourceExists {
		return fmt.Errorf("target cluster key %s not found in SourceCheckConfigs", targetClusterKey.String())
	}

	localClusterKey := cfg.ClusterKey
	triggerIDs, err := manager.Database.GetTriggerIDs(localClusterKey)
	if err != nil {
		return fmt.Errorf("failed to get local trigger IDs: %w", err)
	}

	if len(triggerIDs) == 0 {
		manager.Logger.Info().Msg("No local triggers found for substitution")
		return nil
	}

	count := cfg.Count
	if count > len(triggerIDs) {
		count = len(triggerIDs)
	}

	substitutionSet := make(map[string]struct{})
	for _, id := range triggerIDs[:count] {
		substitutionSet[id] = struct{}{}
	}

	manager.substitutionMutex.Lock()
	manager.substitutionSet = substitutionSet
	manager.substitutionMutex.Unlock()

	manager.Logger.Info().
		Int("count", count).
		String("target_cluster_key", targetClusterKey.String()).
		Msg("Substitution set initialized")

	return nil
}

func (manager *WorkerManager) isInSubstitutionSet(triggerID string) bool {
	manager.substitutionMutex.RLock()
	defer manager.substitutionMutex.RUnlock()
	_, ok := manager.substitutionSet[triggerID]
	return ok
}

func (manager *WorkerManager) getSubstitutionCheckedBy() string {
	cfg := manager.Config.SourceSubstitution
	if cfg == nil {
		return ""
	}
	return cfg.TargetClusterKey.String()
}

func (manager *WorkerManager) makeSourceValidator(clusterKey moira.ClusterKey) (func() error, error) {
	if clusterKey.TriggerSource == moira.GraphiteLocal {
		return manager.validateGraphiteLocal, nil
	}

	source, err := manager.SourceProvider.GetMetricSource(clusterKey)
	if err != nil {
		return nil, err
	}

	return func() error {
		if available, err := source.IsAvailable(); !available {
			return fmt.Errorf("source is not available: %w", err)
		}

		return nil
	}, nil
}

func (manager *WorkerManager) validateGraphiteLocal() error {
	now := time.Now().UTC().Unix()

	if manager.lastData+manager.Config.StopCheckingIntervalSeconds > now {
		return nil
	}

	return fmt.Errorf("graphite local source invalid: no metrics for %d second", now-manager.lastData)
}

func (manager *WorkerManager) startCheckerWorker(w *scheduler) error {
	const maxParallelChecksMaxValue = 1024 * 8
	if w.getMaxParallelChecks() > maxParallelChecksMaxValue {
		return errors.New("MaxParallel" + w.name + "Checks value is too large")
	}

	manager.tomb.Go(w.startTriggerScheduler)
	manager.Logger.Info().Msg(w.name + " scheduler started")

	triggerIdsToCheckChan := manager.pipeTriggerToCheckQueue(
		w.getTriggersToCheck,
		w.getMaxParallelChecks(),
	)

	for i := 0; i < w.getMaxParallelChecks(); i++ {
		manager.tomb.Go(func() error {
			return manager.startTriggerHandler(
				triggerIdsToCheckChan,
				w.metrics,
				w.clusterKey,
			)
		})
	}

	return nil
}

func (manager *WorkerManager) startLazyTriggers() error {
	manager.lastData = time.Now().UTC().Unix()

	manager.lazyTriggerIDs.Store(make(map[string]bool))
	manager.tomb.Go(manager.lazyTriggersWorker)

	manager.tomb.Go(manager.checkTriggersToCheckCount)

	return nil
}

func (manager *WorkerManager) checkTriggersToCheckCount() error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	defer checkTicker.Stop()

	for {
		select {
		case <-manager.tomb.Dying():
			return nil

		case <-checkTicker.C:
			for clusterKey := range manager.Config.SourceCheckConfigs {
				metrics, err := manager.Metrics.GetCheckMetricsBySource(clusterKey)
				if err != nil {
					continue
				}

				triggersToCheck, err := getTriggersToCheck(manager.Database, clusterKey)
				if err != nil {
					continue
				}

				metrics.TriggersToCheckCount.Update(triggersToCheck)
			}
		}
	}
}

func getTriggersToCheck(database moira.Database, clusterKey moira.ClusterKey) (int64, error) {
	return database.GetTriggersToCheckCount(clusterKey)
}

// Stop stops checks triggers.
func (manager *WorkerManager) Stop() error {
	manager.tomb.Kill(nil)
	return manager.tomb.Wait()
}
