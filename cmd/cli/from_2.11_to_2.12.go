package main

import (
	"context"
	"github.com/moira-alert/moira"
)

const contactFetchCount int64 = 10_000

func updateFrom211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.11 -> 2.12 was started")

	ctx := context.Background()
	err := splitNotificationHistoryByContactId(ctx, logger, database, contactFetchCount)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.11 -> 2.12 was finished")
	return nil
}

func downgradeTo211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.11 -> 2.12 started")

	ctx := context.Background()
	err := unionNotificationHistory(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.11 -> 2.12 was finished")
	return nil
}
