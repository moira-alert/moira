package main

import (
	"github.com/moira-alert/moira"
)

func updateFrom211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.11 -> 2.12 was started")

	if err := database.UpdateTelegramUsersRecords(); err != nil {
		return err
	}

	logger.Info().Msg("Update 2.11 -> 2.12 was finished")

	return nil
}

func downgradeTo211(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.12 -> 2.11 started")

	if err := database.DowngradeTelegramUsersRecords(); err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.12 -> 2.11 was finished")

	return nil
}
