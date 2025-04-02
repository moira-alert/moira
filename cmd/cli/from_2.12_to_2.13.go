package main

import (
	"context"

	"github.com/moira-alert/moira"
)

func updateFrom212(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.12 -> 2.13 was started")

	ctx := context.Background()
	err := splitNotificationHistoryByContactID(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.12 -> 2.13 was finished")
	return nil
}

func downgradeTo212(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.13 -> 2.12 started")

	err := mergeNotificationHistory(logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.13 -> 2.12 was finished")
	return nil
}
