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
	return database.CleanUpAbandonedRetentions()
}

func handleCleanUpAbandonedTriggerLastCheck(database moira.Database) error {
	return database.CleanUpAbandonedTriggerLastCheck()
}

func handleRemoveMetricsByPrefix(database moira.Database, prefix string) error {
	return database.RemoveMetricsByPrefix(prefix)
}

func handleRemoveAllMetrics(database moira.Database) error {
	return database.RemoveAllMetrics()
}
