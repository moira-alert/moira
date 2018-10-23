package worker

import (
	"time"

	"github.com/moira-alert/moira/database"
)

const (
	sleepAfterGetTriggersError              = time.Second * 1
	sleepWhenNoTriggersWithoutSubscriptions = time.Second * 2
	triggersWithoutSubscriptionsTicker      = time.Second * 10
)

func (worker *Checker) runTriggersWithoutSubscriptionsUpdater() error {
	checkTicker := time.NewTicker(triggersWithoutSubscriptionsTicker)
	worker.Logger.Info("Start tracking triggers without subscriptions")
	for {
		select {
		case <-worker.tomb.Dying():
			checkTicker.Stop()
			worker.Logger.Info("Tracking triggers without subscriptions stopped")
			return nil
		case <-checkTicker.C:
			err := worker.fillTriggersWithoutSubscriptions()
			if err == database.ErrNil {
				<-time.After(sleepWhenNoTriggersWithoutSubscriptions)
			} else {
				worker.Logger.Errorf("Failed to get triggers without subscriptions: %s", err.Error())
				<-time.After(sleepAfterGetTriggersError)
			}
		}
	}
}

func (worker *Checker) fillTriggersWithoutSubscriptions() error {
	triggerIDs, err := worker.Database.GetTriggersWithoutSubscriptions()
	if err != nil {
		return err
	}
	worker.triggersWithoutSubscriptions = make(map[string]bool)
	for _, triggerID := range triggerIDs {
		worker.triggersWithoutSubscriptions[triggerID] = true
	}
	return nil
}
