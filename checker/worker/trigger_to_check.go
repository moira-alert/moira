package worker

import (
	"time"
)

const sleepAfterGetTriggerIDError = time.Second * 1
const sleepWhenNoTriggerToCheck = time.Millisecond * 500

func (worker *Checker) startTriggerToCheckGetter(fetch func(int) ([]string, error), batchSize int) <-chan string {
	triggerIDsToCheck := make(chan string, batchSize*2) //nolint
	worker.tomb.Go(func() error { return worker.triggerToCheckGetter(fetch, batchSize, triggerIDsToCheck) })
	return triggerIDsToCheck
}

func (worker *Checker) triggerToCheckGetter(fetch func(int) ([]string, error), batchSize int, triggerIDsToCheck chan<- string) error {
	var fetchDelay time.Duration
	for {
		startFetch := time.After(fetchDelay)
		select {
		case <-worker.tomb.Dying():
			close(triggerIDsToCheck)
			return nil
		case <-startFetch:
			triggerIDs, err := fetch(batchSize)
			fetchDelay = worker.handleFetchResponse(triggerIDs, err, triggerIDsToCheck)
		}
	}
}

func (worker *Checker) handleFetchResponse(triggerIDs []string, fetchError error, triggerIDsToCheck chan<- string) time.Duration {
	if fetchError != nil {
		worker.Logger.Errorb().
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
