package main

import (
	"time"

	"github.com/moira-alert/moira"
)

func cleanUpOutdatedMetrics(config cleanupConfig, database moira.Database) error {
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

func cleanUpAbandonedTriggerLastCheck(database moira.Database) error {
	return database.CleanUpAbandonedTriggerLastCheck()
}
