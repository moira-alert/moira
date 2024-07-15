package main

import (
	"context"

	"github.com/moira-alert/moira"
)

func updateFrom212(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.11 -> 2.12 was started")

	ctx := context.Background()
	err := splitNotificationHistoryByContactId(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.11 -> 2.12 was finished")
	return nil
}

func downgradeTo212(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.11 -> 2.12 started")

	ctx := context.Background()
	err := mergeNotificationHistory(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.11 -> 2.12 was finished")
	return nil
}
