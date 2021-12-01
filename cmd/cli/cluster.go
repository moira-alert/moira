package main

import (
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

var noSuchKeyError = "ERR no such key"
var anyTagsSubscriptionsKeyOld = "moira-any-tags-subscriptions"
var anyTagsSubscriptionsKeyNew = "{moira-tag-subscriptions}:moira-any-tags-subscriptions"
var triggersListKeyOld = "moira-triggers-list"
var triggersListKeyNew = "{moira-triggers-list}:moira-triggers-list"
var remoteTriggersListKeyOld = "moira-remote-triggers-list"
var remoteTriggersListKeyNew = "{moira-triggers-list}:moira-remote-triggers-list"
var tagSubscriptionsKeyPrefixOld = "moira-tag-subscriptions:"
var tagSubscriptionsKeyPrefixNew = "{moira-tag-subscriptions}:"
var tagTriggersKeyKeyPrefixOld = "moira-tag-triggers:"
var tagTriggersKeyKeyPrefixNew = "{moira-tag-triggers}:"

func renameKey(database moira.Database, oldKey string, newKey string) error {
	switch d := database.(type) {
	case *redis.DbConnector:
		err := d.Client().Rename(d.Context(), oldKey, newKey).Err()
		if err != nil {
			if err.Error() != noSuchKeyError {
				return err
			}
		}
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
	}

	return nil
}

func renameAnyTagsSubscriptionsKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, anyTagsSubscriptionsKeyOld, anyTagsSubscriptionsKeyNew)
	if err != nil {
		return err
	}

	logger.Info("renameAnyTagsSubscriptionsKeyForwards done")

	return nil
}

func renameAnyTagsSubscriptionsKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, anyTagsSubscriptionsKeyNew, anyTagsSubscriptionsKeyOld)
	if err != nil {
		return err
	}

	logger.Info("renameAnyTagsSubscriptionsKeyReverse done")

	return nil
}

func renameTriggersListKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, triggersListKeyOld, triggersListKeyNew)
	if err != nil {
		return err
	}

	logger.Info("renameTriggersListKeyForwards done")

	return nil
}

func renameTriggersListKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, triggersListKeyNew, triggersListKeyOld)
	if err != nil {
		return err
	}

	logger.Info("renameTriggersListKeyReverse done")

	return nil
}

func renameRemoteTriggersListKeyForwards(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, remoteTriggersListKeyOld, remoteTriggersListKeyNew)
	if err != nil {
		return err
	}

	logger.Info("renameRemoteTriggersListKeyForwards done")

	return nil
}

func renameRemoteTriggersListKeyReverse(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, remoteTriggersListKeyNew, remoteTriggersListKeyOld)
	if err != nil {
		return err
	}

	logger.Info("renameRemoteTriggersListKeyReverse done")

	return nil
}

func renameTagSubscriptionsKeyForwards(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagSubscriptionsKeyPrefixOld, tagSubscriptionsKeyPrefixNew)
	if err != nil {
		return err
	}

	logger.Info("renameTagSubscriptionsKeyForwards done")

	return nil
}

func renameTagSubscriptionsKeyReverse(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagSubscriptionsKeyPrefixNew, tagSubscriptionsKeyPrefixOld)
	if err != nil {
		return err
	}

	logger.Info("renameTagSubscriptionsKeyReverse done")

	return nil
}

func renameTagTriggersKeyKeyForwards(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagTriggersKeyKeyPrefixOld, tagTriggersKeyKeyPrefixNew)
	if err != nil {
		return err
	}

	logger.Info("renameTagTriggersKeyKeyForwards done")

	return nil
}

func renameTagTriggersKeyKeyReverse(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, tagTriggersKeyKeyPrefixNew, tagTriggersKeyKeyPrefixOld)
	if err != nil {
		return err
	}

	logger.Info("renameTagTriggersKeyKeyReverse done")

	return nil
}

func moveToClusterForwards(logger moira.Logger, database moira.Database) error {
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

	logger.Info("moveToClusterForwards done")

	return nil
}

func moveToClusterReverse(logger moira.Logger, database moira.Database) error {
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

	logger.Info("moveToClusterReverse done")

	return nil
}
