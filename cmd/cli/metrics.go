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

	if err = database.CleanUpOutdatedMetrics(duration); err != nil {
		return err
	}

	return nil
}

func handleCleanUpFutureMetrics(config cleanupConfig, database moira.Database) error {
	duration, err := time.ParseDuration(config.CleanupFutureMetricsDuration)
	if err != nil {
		return err
	}

	if err = database.CleanUpFutureMetrics(duration); err != nil {
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

func handleCleanUpAbandonedTags(database moira.Database) (int, error) {
	return database.CleanUpAbandonedTags()
}

func handleRemoveMetricsByPrefix(database moira.Database, prefix string) error {
	return database.RemoveMetricsByPrefix(prefix)
}

func handleRemoveAllMetrics(database moira.Database) error {
	return database.RemoveAllMetrics()
}

func handleCleanUpOutdatedPatternMetrics(database moira.Database) (int64, error) {
	return database.CleanupOutdatedPatternMetrics()
}
