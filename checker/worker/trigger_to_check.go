package worker

import (
	"time"

	"github.com/patrickmn/go-cache"
)

const sleepAfterGetTriggerIDError = time.Second * 1
const sleepWhenNoTriggerToCheck = time.Millisecond * 500

func (check *Checker) startTriggerToCheckGetter(fetch func(int) ([]string, error), batchSize int) <-chan string {
	triggerIDsToCheck := make(chan string, batchSize*2) //nolint
	check.tomb.Go(func() error {
		return check.triggerToCheckGetter(fetch, batchSize, triggerIDsToCheck)
	})
	return triggerIDsToCheck
}

func (check *Checker) triggerToCheckGetter(fetch func(int) ([]string, error), batchSize int, triggerIDsToCheck chan<- string) error {
	var fetchDelay time.Duration
	for {
		startFetch := time.After(fetchDelay)
		select {
		case <-check.tomb.Dying():
			close(triggerIDsToCheck)
			return nil
		case <-startFetch:
			triggerIDs, err := fetch(batchSize)
			fetchDelay = check.handleFetchResponse(triggerIDs, err, triggerIDsToCheck)
		}
	}
}

func (check *Checker) handleFetchResponse(triggerIDs []string, fetchError error, triggerIDsToCheck chan<- string) time.Duration {
	if fetchError != nil {
		check.Logger.Error().
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

func (check *Checker) filterOutLazyTriggerIDs(triggerIDs []string) []string {
	triggerIDsToCheck := make([]string, 0, len(triggerIDs))

	lazyTriggerIDs := check.lazyTriggerIDs.Load().(map[string]bool)

	for _, triggerID := range triggerIDs {
		if _, ok := lazyTriggerIDs[triggerID]; ok {
			randomDuration := check.getRandomLazyCacheDuration()
			cacheContainsIdErr := check.LazyTriggersCache.Add(triggerID, true, randomDuration)

			if cacheContainsIdErr != nil {
				continue
			}
		}

		cacheContainsIdErr := check.TriggerCache.Add(triggerID, true, cache.DefaultExpiration)
		if cacheContainsIdErr == nil {
			triggerIDsToCheck = append(triggerIDsToCheck, triggerID)
		}
	}

	return triggerIDsToCheck
}
