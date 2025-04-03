package main

import "github.com/moira-alert/moira"

func updateFrom213(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.13 -> 2.14 started")

	err := fillTeamNamesHash(logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.13 -> 2.14 was finished")

	return nil
}

func downgradeTo213(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.14 -> 2.13 started")

	err := removeTeamNamesHash(logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.14 -> 2.13 was finished")

	return nil
}
