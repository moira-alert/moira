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

func (check *Checker) lazyTriggersWorker() error {
	localConfig, ok := check.Config.SourceCheckConfigs[moira.MakeClusterKey(moira.GraphiteLocal, "default")]
	if !ok {
		return fmt.Errorf("can not initialize lazyTriggersWorker: default local source is not configured")
	}

	if check.Config.LazyTriggersCheckInterval <= localConfig.CheckInterval {
		check.Logger.Info().
			Interface("lazy_triggers_check_interval", check.Config.LazyTriggersCheckInterval).
			Interface("check_interval", localConfig.CheckInterval).
			Msg("Lazy triggers worker won't start because lazy triggers interval is less or equal to check interval")
		return nil
	}
	checkTicker := time.NewTicker(lazyTriggersWorkerTicker)
	check.Logger.Info().
		Interface("lazy_triggers_check_interval", check.Config.LazyTriggersCheckInterval).
		Interface("update_lazy_triggers_every", lazyTriggersWorkerTicker).
		Msg("Start lazy triggers worker")

	for {
		select {
		case <-check.tomb.Dying():
			checkTicker.Stop()
			check.Logger.Info().Msg("Lazy triggers worker stopped")
			return nil
		case <-checkTicker.C:
			err := check.fillLazyTriggerIDs()
			if err != nil {
				check.Logger.Error().
					Error(err).
					Msg("Failed to get lazy triggers")
			}
		}
	}
}

func (check *Checker) fillLazyTriggerIDs() error {
	triggerIDs, err := check.Database.GetUnusedTriggerIDs()
	if err != nil {
		return err
	}
	newLazyTriggerIDs := make(map[string]bool)
	for _, triggerID := range triggerIDs {
		newLazyTriggerIDs[triggerID] = true
	}
	check.lazyTriggerIDs.Store(newLazyTriggerIDs)
	check.Metrics.UnusedTriggersCount.Update(int64(len(newLazyTriggerIDs)))
	return nil
}

func (check *Checker) getRandomLazyCacheDuration() time.Duration {
	maxLazyCacheSeconds := check.Config.LazyTriggersCheckInterval.Seconds()
	min := maxLazyCacheSeconds / 2 //nolint
	i := rand.Float64()*min + min
	return time.Duration(i) * time.Second
}
