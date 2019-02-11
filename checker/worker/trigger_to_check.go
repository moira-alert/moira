package worker

import (
	"time"
)

const sleepAfterGetTriggerIDError = time.Second * 1
const sleepWhenNoTriggerToCheck = time.Millisecond * 500

func (worker *Checker) startTriggerToCheckGetter(isRemote bool, batchSize int) <-chan string {
	triggerIDsToCheck := make(chan string, batchSize*2)
	worker.tomb.Go(func() error { return worker.triggerToCheckGetter(isRemote, batchSize, triggerIDsToCheck) })
	return triggerIDsToCheck
}

func (worker *Checker) triggerToCheckGetter(isRemote bool, batchSize int, triggerIDsToCheck chan<- string) error {
	for {
		select {
		case <-worker.tomb.Dying():
			close(triggerIDsToCheck)
			return nil
		default:
			triggerIDs, err := worker.getTriggersToCheck(isRemote, batchSize)
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

func (worker *Checker) getTriggersToCheck(isRemote bool, batchSize int) ([]string, error) {
	if isRemote {
		return worker.Database.GetRemoteTriggersToCheck(batchSize)
	}
	return worker.Database.GetTriggersToCheck(batchSize)
}
