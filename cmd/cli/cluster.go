package main

import (
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

var (
	anyTagsSubscriptionsKeyOld   = "moira-any-tags-subscriptions"
	anyTagsSubscriptionsKeyNew   = "{moira-tag-subscriptions}:moira-any-tags-subscriptions"
	triggersListKeyOld           = "moira-triggers-list"
	triggersListKeyNew           = "{moira-triggers-list}:moira-triggers-list"
	remoteTriggersListKeyOld     = "moira-remote-triggers-list"
	remoteTriggersListKeyNew     = "{moira-triggers-list}:moira-remote-triggers-list"
	tagSubscriptionsKeyPrefixOld = "moira-tag-subscriptions:"
	tagSubscriptionsKeyPrefixNew = "{moira-tag-subscriptions}:"
	tagTriggersKeyKeyPrefixOld   = "moira-tag-triggers:"
	tagTriggersKeyKeyPrefixNew   = "{moira-tag-triggers}:"
)

func renameKey(database moira.Database, oldValue, newValue string) error {
	switch d := database.(type) {
	case *redis.DbConnector:
		pipe := d.Client().TxPipeline()

		iter := d.Client().Scan(d.Context(), 0, oldValue, 0).Iterator()
		for iter.Next(d.Context()) {
			oldKey := iter.Val()
			newKey := strings.Replace(iter.Val(), oldValue, newValue, 1)
			pipe.Rename(d.Context(), oldKey, newKey)
		}

		_, err := pipe.Exec(d.Context())
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	return nil
}

func changeKeysPrefix(database moira.Database, oldPrefix string, newPrefix string) error {
	switch d := database.(type) {
	case *redis.DbConnector:
		pipe := d.Client().TxPipeline()

		iter := d.Client().Scan(d.Context(), 0, oldPrefix+"*", 0).Iterator()
		for iter.Next(d.Context()) {
			oldKey := iter.Val()
			newKey := strings.Replace(iter.Val(), oldPrefix, newPrefix, 1)
			pipe.Rename(d.Context(), oldKey, newKey)
		}

		_, err := pipe.Exec(d.Context())
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	return nil
}

func renameAnyTagsSubscriptionsKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, anyTagsSubscriptionsKeyOld, anyTagsSubscriptionsKeyNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameAnyTagsSubscriptionsKeyForwards done")

	return nil
}

func renameAnyTagsSubscriptionsKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, anyTagsSubscriptionsKeyNew, anyTagsSubscriptionsKeyOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameAnyTagsSubscriptionsKeyReverse done")

	return nil
}

func renameTriggersListKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, triggersListKeyOld, triggersListKeyNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTriggersListKeyForwards done")

	return nil
}

func renameTriggersListKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, triggersListKeyNew, triggersListKeyOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTriggersListKeyReverse done")

	return nil
}

func renameRemoteTriggersListKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, remoteTriggersListKeyOld, remoteTriggersListKeyNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameRemoteTriggersListKeyForwards done")

	return nil
}

func renameRemoteTriggersListKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, remoteTriggersListKeyNew, remoteTriggersListKeyOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameRemoteTriggersListKeyReverse done")

	return nil
}

func renameTagSubscriptionsKeyForwards(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagSubscriptionsKeyPrefixOld, tagSubscriptionsKeyPrefixNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTagSubscriptionsKeyForwards done")

	return nil
}

func renameTagSubscriptionsKeyReverse(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagSubscriptionsKeyPrefixNew, tagSubscriptionsKeyPrefixOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTagSubscriptionsKeyReverse done")

	return nil
}

func renameTagTriggersKeyKeyForwards(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagTriggersKeyKeyPrefixOld, tagTriggersKeyKeyPrefixNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTagTriggersKeyKeyForwards done")

	return nil
}

func renameTagTriggersKeyKeyReverse(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagTriggersKeyKeyPrefixNew, tagTriggersKeyKeyPrefixOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("renameTagTriggersKeyKeyReverse done")

	return nil
}

func addRedisClusterSupport(logger moira.Logger, database moira.Database) error {
	err := renameAnyTagsSubscriptionsKeyForwards(logger, database)
	if err != nil {
		return err
	}

	err = renameTriggersListKeyForwards(logger, database)
	if err != nil {
		return err
	}

	err = renameRemoteTriggersListKeyForwards(logger, database)
	if err != nil {
		return err
	}

	err = renameTagSubscriptionsKeyForwards(logger, database)
	if err != nil {
		return err
	}

	err = renameTagTriggersKeyKeyForwards(logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("addRedisClusterSupport done")

	return nil
}

func removeRedisClusterSupport(logger moira.Logger, database moira.Database) error {
	err := renameAnyTagsSubscriptionsKeyReverse(logger, database)
	if err != nil {
		return err
	}

	err = renameTriggersListKeyReverse(logger, database)
	if err != nil {
		return err
	}

	err = renameRemoteTriggersListKeyReverse(logger, database)
	if err != nil {
		return err
	}

	err = renameTagSubscriptionsKeyReverse(logger, database)
	if err != nil {
		return err
	}

	err = renameTagTriggersKeyKeyReverse(logger, database)
	if err != nil {
		return err
	}

	logger.Info().Msg("removeRedisClusterSupport done")

	return nil
}
