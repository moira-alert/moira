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
	logger.Infob().
		Int("triggers_to_delete", len(triggers)).
		String("prefix", prefix).
		String("delay", delay.String()).
		Msg("Triggers that start with given prefix would be removed after delay")
	logger.Infob().Msg("You can cancel execution by Ctrl+C")
	time.Sleep(delay)

	logger.Infob().
		String("prefix", prefix).
		Msg("Removing triggers start with given prefix has started")

	for _, id := range triggers {
		err := database.RemoveTrigger(id)
		if err != nil {
			return fmt.Errorf("can't remove trigger with id %s: %w", id, err)
		}
	}
	logger.Infob().
		String("prefix", prefix).
		Int("deleted_triggers_count", len(triggers)).
		Value("deleted_triggers", triggers).
		Msg("Removing triggers start with given prefix has finished")

	return nil
}
