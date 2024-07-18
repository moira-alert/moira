package worker

import (
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	sleepAfterGetTriggerIDError = time.Second * 1
	sleepWhenNoTriggerToCheck   = time.Millisecond * 500
)

func (manager *Manager) pipeTriggerToCheckQueue(fetch func(int) ([]string, error), batchSize int) <-chan string {
	triggerIDsToCheck := make(chan string, batchSize*2) //nolint
	manager.tomb.Go(func() error {
		return manager.pipeTriggerToCheckQueueToChan(fetch, batchSize, triggerIDsToCheck)
	})
	return triggerIDsToCheck
}

func (manager *Manager) pipeTriggerToCheckQueueToChan(fetch func(int) ([]string, error), batchSize int, triggerIDsToCheck chan<- string) error {
	var fetchDelay time.Duration
	for {
		startFetch := time.After(fetchDelay)

		select {
		case <-manager.tomb.Dying():
			close(triggerIDsToCheck)
			return nil

		case <-startFetch:
			triggerIDs, err := fetch(batchSize)
			fetchDelay = manager.handleFetchResponse(triggerIDs, err, triggerIDsToCheck)
		}
	}
}

func (manager *Manager) handleFetchResponse(triggerIDs []string, fetchError error, triggerIDsToCheck chan<- string) time.Duration {
	if fetchError != nil {
		manager.Logger.Error().
			Error(fetchError).
			Msg("Failed to handle trigger loop")

		return sleepAfterGetTriggerIDError
	}

	if len(triggerIDs) == 0 {
		return sleepWhenNoTriggerToCheck
	}

	for _, triggerID := range triggerIDs {
		triggerIDsToCheck <- triggerID
	}

	return time.Duration(0)
}

func (manager *Manager) filterOutLazyTriggerIDs(triggerIDs []string) []string {
	triggerIDsToCheck := make([]string, 0, len(triggerIDs))

	lazyTriggerIDs := manager.lazyTriggerIDs.Load().(map[string]bool)

	for _, triggerID := range triggerIDs {
		if _, ok := lazyTriggerIDs[triggerID]; ok {
			randomDuration := manager.getRandomLazyCacheDuration()
			cacheContainsIDErr := manager.LazyTriggersCache.Add(triggerID, true, randomDuration)

			if cacheContainsIDErr != nil {
				continue
			}
		}

		cacheContainsIDErr := manager.TriggerCache.Add(triggerID, true, cache.DefaultExpiration)
		if cacheContainsIDErr == nil {
			triggerIDsToCheck = append(triggerIDsToCheck, triggerID)
		}
	}

	return triggerIDsToCheck
}
