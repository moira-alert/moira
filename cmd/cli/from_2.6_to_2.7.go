package main

import "github.com/moira-alert/moira"

func updateFrom26(logger moira.Logger, dataBase moira.Database) error {
	logger.Infob().Msg("Update 2.6 -> 2.7 was started")

	logger.Infob().Msg("Adding Redis Cluster support was started")
	if err := addRedisClusterSupport(logger, dataBase); err != nil {
		return err
	}

	logger.Infob().Msg("Update 2.6 -> 2.7 was finished")
	return nil
}

func downgradeTo26(logger moira.Logger, dataBase moira.Database) error {
	logger.Infob().Msg("Downgrade 2.7 -> 2.6 started")

	logger.Infob().Msg("Removing Redis Cluster support was started")
	if err := removeRedisClusterSupport(logger, dataBase); err != nil {
		return err
	}

	logger.Infob().Msg("Downgrade 2.7 -> 2.6 was finished")
	return nil
}
