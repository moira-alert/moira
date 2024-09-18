package worker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/moira-alert/moira"
)

const (
	lazyTriggersWorkerTicker = time.Second * 10
)

func (manager *WorkerManager) lazyTriggersWorker() error {
	localConfig, ok := manager.Config.SourceCheckConfigs[moira.DefaultLocalCluster]
	if !ok {
		return fmt.Errorf("can not initialize lazyTriggersWorker: default local source is not configured")
	}

	if manager.Config.LazyTriggersCheckInterval <= localConfig.CheckInterval {
		manager.Logger.Info().
			Interface("lazy_triggers_check_interval", manager.Config.LazyTriggersCheckInterval).
			Interface("check_interval", localConfig.CheckInterval).
			Msg("Lazy triggers worker won't start because lazy triggers interval is less or equal to check interval")
		return nil
	}
	checkTicker := time.NewTicker(lazyTriggersWorkerTicker)
	manager.Logger.Info().
		Interface("lazy_triggers_check_interval", manager.Config.LazyTriggersCheckInterval).
		Interface("update_lazy_triggers_every", lazyTriggersWorkerTicker).
		Msg("Start lazy triggers worker")

	for {
		select {
		case <-manager.tomb.Dying():
			checkTicker.Stop()
			manager.Logger.Info().Msg("Lazy triggers worker stopped")
			return nil
		case <-checkTicker.C:
			err := manager.fillLazyTriggerIDs()
			if err != nil {
				manager.Logger.Error().
					Error(err).
					Msg("Failed to get lazy triggers")
			}
		}
	}
}

func (manager *WorkerManager) fillLazyTriggerIDs() error {
	triggerIDs, err := manager.Database.GetUnusedTriggerIDs()
	if err != nil {
		return err
	}
	newLazyTriggerIDs := make(map[string]bool)
	for _, triggerID := range triggerIDs {
		newLazyTriggerIDs[triggerID] = true
	}
	manager.lazyTriggerIDs.Store(newLazyTriggerIDs)
	manager.Metrics.UnusedTriggersCount.Update(int64(len(newLazyTriggerIDs)))
	return nil
}

func (manager *WorkerManager) getRandomLazyCacheDuration() time.Duration {
	maxLazyCacheSeconds := manager.Config.LazyTriggersCheckInterval.Seconds()
	min := maxLazyCacheSeconds / 2 //nolint
	i := rand.Float64()*min + min
	return time.Duration(i) * time.Second
}
