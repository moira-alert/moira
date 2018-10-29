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
		worker.Logger.Infof("Lazy triggers worker won't start because lazy triggers interval '%v' is less or equal to check interval '%v'", worker.Config.LazyTriggersCheckInterval, worker.Config.CheckInterval)
		return nil
	}
	checkTicker := time.NewTicker(lazyTriggersWorkerTicker)
	worker.Logger.Infof("Start lazy triggers worker. Update lazy triggers list every %v", lazyTriggersWorkerTicker)
	worker.Logger.Infof("Check lazy triggers every %v", worker.Config.LazyTriggersCheckInterval)
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Lazy triggers worker stopped")
			return nil
		case <-checkTicker.C:
			err := worker.fillLazyTriggerIDs()
			if err != nil {
				worker.Logger.Errorf("Failed to get lazy triggers: %s", err.Error())
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
	worker.lazyTriggerIDs = newLazyTriggerIDs
	worker.Metrics.UnusedTriggersCount.Update(int64(len(worker.lazyTriggerIDs)))
	return nil
}

func (worker *Checker) getRandomLazyCacheDuration() time.Duration {
	maxLazyCacheSeconds := worker.Config.LazyTriggersCheckInterval.Seconds()
	min := maxLazyCacheSeconds / 2
	i := rand.Float64()*min + min
	return time.Duration(i) * time.Second
}
