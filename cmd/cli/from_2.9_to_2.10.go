package main

import (
	"context"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

func updateFrom29(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Update 2.9 -> 2.10 was started")

	ctx := context.Background()

	err := createKeyForLocalTriggers(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Update 2.9 -> 2.10 was finished")

	return nil
}

func downgradeTo29(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Downgrade 2.10 -> 2.9 started")

	ctx := context.Background()

	err := revertCreateKeyForLocalTriggers(ctx, logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.10 -> 2.9 was finished")

	return nil
}

var (
	triggersListKey           = "{moira-triggers-list}:moira-triggers-list"
	localTriggersListKey      = "{moira-triggers-list}:moira-local-triggers-list"
	remoteTriggersListKey     = "{moira-triggers-list}:moira-remote-triggers-list"
	prometheusTriggersListKey = "{moira-triggers-list}:moira-prometheus-triggers-list"
)

func createKeyForLocalTriggers(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start createKeyForLocalTriggers")

	switch d := database.(type) {
	case *redis.DbConnector:
		client := d.Client()

		localTriggerIds, err := client.SDiff(ctx, triggersListKey, remoteTriggersListKey, prometheusTriggersListKey).Result()
		if err != nil {
			return err
		}

		logger.Info().Msg("Finish getting local trigger IDs")

		_, err = client.SAdd(ctx, localTriggersListKey, localTriggerIds).Result()
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished createKeyForLocalTriggers")

	return nil
}

func revertCreateKeyForLocalTriggers(ctx context.Context, logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start revertCreateKeyForLocalTriggers")

	switch d := database.(type) {
	case *redis.DbConnector:
		client := d.Client()

		err := client.Del(ctx, localTriggersListKey).Err()
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	logger.Info().Msg("Successfully finished revertCreateKeyForLocalTriggers")

	return nil
}
