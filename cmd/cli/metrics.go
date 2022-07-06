package main

import (
	"time"

	"github.com/moira-alert/moira"
)

func handleCleanUpOutdatedMetrics(config cleanupConfig, database moira.Database) error {
	duration, err := time.ParseDuration(config.CleanupMetricsDuration)
	if err != nil {
		return err
	}

	err = database.CleanUpOutdatedMetrics(duration)
	if err != nil {
		return err
	}

	return nil
}

func handleCleanUpAbandonedRetentions(database moira.Database) error {
	err := database.CleanUpAbandonedRetentions()
	if err != nil {
		return err
	}

	return nil
}

func handleCleanUpAbandonedTriggerLastCheck(database moira.Database) error {
	return database.CleanUpAbandonedTriggerLastCheck()
}

func handleRemoveMetricsByPrefix(database moira.Database, prefix string) error {
	err := database.RemoveMetricsByPrefix(prefix)
	if err != nil {
		return err
	}

	return nil
}

func handleRemoveAllMetrics(database moira.Database) error {
	err := database.RemoveAllMetrics()
	if err != nil {
		return err
	}

	return nil
}
