package main

import (
	"github.com/moira-alert/moira"
	"time"
)

func cleanupOutdatedMetrics(config cleanupConfig, database moira.Database) error {
	duration, err := time.ParseDuration(config.CleanupMetricsDuration)
	if err != nil {
		return err
	}

	err = database.CleanupOutdatedMetrics(duration)
	if err != nil {
		return err
	}

	return nil
}
