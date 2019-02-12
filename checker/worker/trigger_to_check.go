package worker

import (
	"time"
)

const sleepAfterGetTriggerIDError = time.Second * 1
const sleepWhenNoTriggerToCheck = time.Millisecond * 500

func (worker *Checker) startTriggerToCheckGetter(fetch func(int) ([]string, error), batchSize int) <-chan string {
	triggerIDsToCheck := make(chan string, batchSize*2)
	worker.tomb.Go(func() error { return worker.triggerToCheckGetter(fetch, batchSize, triggerIDsToCheck) })
	return triggerIDsToCheck
}

func (worker *Checker) triggerToCheckGetter(fetch func(int) ([]string, error), batchSize int, triggerIDsToCheck chan<- string) error {
	for {
		select {
		case <-worker.tomb.Dying():
			close(triggerIDsToCheck)
			return nil
		default:
			triggerIDs, err := fetch(batchSize)
			if err != nil {
				worker.Logger.Errorf("Failed to handle trigger loop: %s", err.Error())
				<-time.After(sleepAfterGetTriggerIDError)
				continue
			}
			if len(triggerIDs) == 0 {
				<-time.After(sleepWhenNoTriggerToCheck)
				continue
			}
			for _, triggerID := range triggerIDs {
				triggerIDsToCheck <- triggerID
			}
		}
	}
}
