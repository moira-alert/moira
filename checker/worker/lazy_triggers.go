package worker

import (
	"math/rand"
	"time"
)

const (
	lazyTriggersWorkerTicker = time.Second * 10
)

func (worker *Checker) lazyTriggersWorker() error {
	if worker.Config.LazyTriggersCheckInterval <= worker.Config.CheckInterval {
		worker.Logger.Infob().
			Value("lazy_triggers_check_interval", worker.Config.LazyTriggersCheckInterval).
			Value("check_interval", worker.Config.CheckInterval).
			Msg("Lazy triggers worker won't start because lazy triggers interval is less or equal to check interval")
		return nil
	}
	checkTicker := time.NewTicker(lazyTriggersWorkerTicker)
	worker.Logger.Infob().
		Value("lazy_triggers_check_interval", worker.Config.LazyTriggersCheckInterval).
		Value("update_lazy_triggers_every", lazyTriggersWorkerTicker).
		Msg("Start lazy triggers worker")

	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Lazy triggers worker stopped")
			return nil
		case <-checkTicker.C:
			err := worker.fillLazyTriggerIDs()
			if err != nil {
				worker.Logger.ErrorWithError("Failed to get lazy triggers", err)
			}
		}
	}
}

func (worker *Checker) fillLazyTriggerIDs() error {
	triggerIDs, err := worker.Database.GetUnusedTriggerIDs()
	if err != nil {
		return err
	}
	newLazyTriggerIDs := make(map[string]bool)
	for _, triggerID := range triggerIDs {
		newLazyTriggerIDs[triggerID] = true
	}
	worker.lazyTriggerIDs.Store(newLazyTriggerIDs)
	worker.Metrics.UnusedTriggersCount.Update(int64(len(newLazyTriggerIDs)))
	return nil
}

func (worker *Checker) getRandomLazyCacheDuration() time.Duration {
	maxLazyCacheSeconds := worker.Config.LazyTriggersCheckInterval.Seconds()
	min := maxLazyCacheSeconds / 2 //nolint
	i := rand.Float64()*min + min
	return time.Duration(i) * time.Second
}
