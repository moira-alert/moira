package worker

import (
	"math/rand"
	"time"
)

const (
	lazyTriggersWorkerTicker = time.Second * 10
	maxLazyCacheSeconds      = float64(10 * 60)
)

func (worker *Checker) lazyTriggersWorker() error {
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

func getRandomLazyCacheDuration() time.Duration {
	min := maxLazyCacheSeconds / 2
	i := rand.Float64()*min + min
	return time.Duration(i) * time.Second
}
