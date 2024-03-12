package main

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

// Added delay because command is potentially dangerous and can delete unwanted triggers.
var delay = 10 * time.Second

func handleRemoveTriggersStartWith(logger moira.Logger, database moira.Database, prefix string) error {
	triggers, err := database.GetTriggerIDsStartWith(prefix)
	if err != nil {
		return fmt.Errorf("can't get trigger IDs start with prefix %s: %w", prefix, err)
	}

	return deleteTriggers(logger, triggers, prefix, database)
}

func handleRemoveUnusedTriggersStartWith(logger moira.Logger, database moira.Database, prefix string) error {
	triggers, err := database.GetTriggerIDsStartWith(prefix)
	if err != nil {
		return fmt.Errorf("can't get trigger IDs start with prefix %s: %w", prefix, err)
	}
	unusedTriggers, err := database.GetUnusedTriggerIDs()
	if err != nil {
		return fmt.Errorf("can't get unused trigger IDs; err: %w", err)
	}
	unusedTriggersMap := map[string]struct{}{}

	for _, id := range unusedTriggers {
		unusedTriggersMap[id] = struct{}{}
	}

	triggersToDelete := make([]string, 0)
	for _, id := range triggers {
		if _, ok := unusedTriggersMap[id]; ok {
			triggersToDelete = append(triggersToDelete, id)
		}
	}

	return deleteTriggers(logger, triggersToDelete, prefix, database)
}

func deleteTriggers(logger moira.Logger, triggers []string, prefix string, database moira.Database) error {
	logger.Info().
		Int("triggers_to_delete", len(triggers)).
		String("prefix", prefix).
		String("delay", delay.String()).
		Msg("Triggers that start with given prefix would be removed after delay")
	logger.Info().Msg("You can cancel execution by Ctrl+C")
	time.Sleep(delay)

	logger.Info().
		String("prefix", prefix).
		Msg("Removing triggers start with given prefix has started")

	deletedTriggersCount := 0
	for _, id := range triggers {
		err := database.RemoveTrigger(id)
		if err != nil {
			return fmt.Errorf("can't remove trigger with id %s: %w", id, err)
		}
		deletedTriggersCount++
	}
	logger.Info().
		String("prefix", prefix).
		Int("deleted_triggers_count", len(triggers)).
		Interface("deleted_triggers", triggers).
		Msg("Removing triggers start with given prefix has finished")

	return nil
}
