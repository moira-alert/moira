package main

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

func handleRemoveTriggersStartWith(logger moira.Logger, database moira.Database, prefix string) error {
	triggers, err := database.GetTriggerIDsStartWith(prefix)
	if err != nil {
		return fmt.Errorf("can't get trigger IDs start with prefix %s: %w", prefix, err)
	}

	// Added delay because command is potentially dangerous and can delete unwanted triggers
	const delay = 10 * time.Second
	logger.Infof("%d triggers start with %s would be removed after %s", len(triggers), prefix, delay)
	logger.Info("You can cancel execution by Ctrl+C")
	time.Sleep(delay)

	logger.Infof("Removing triggers start with prefix %s has started", prefix)
	for _, id := range triggers {
		err := database.RemoveTrigger(id)
		if err != nil {
			return fmt.Errorf("can't remove trigger with id %s: %w", id, err)
		}
	}
	logger.Infof("Removing triggers start with prefix %s has finished", prefix)
	logger.Infof("Count of deleted is %d", len(triggers))
	logger.Infof("Removed triggers: %s", triggers)

	return nil
}
