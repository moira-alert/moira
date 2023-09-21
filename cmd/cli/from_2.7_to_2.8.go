package main

import (
	"fmt"

	"github.com/moira-alert/moira"
)

func updateFrom27(logger moira.Logger, dataBase moira.Database) error {
	logger.Info().Msg("Update 2.7 -> 2.8 was started")

	logger.Info().Msg("Rename keys was started")
	if err := updateSubscriptionKeyForAnonymous(logger, dataBase); err != nil {
		return fmt.Errorf("сannot call updateSubscriptionKeyForAnonymous, has error %v", err)
	}

	if err := updateContactKeyForAnonymous(logger, dataBase); err != nil {
		return fmt.Errorf("сannot call updateContactKeyForAnonymous, has error %v", err)
	}

	logger.Info().Msg("Update 2.7 -> 2.8 was finished")
	return nil
}

func downgradeTo27(logger moira.Logger, dataBase moira.Database) error {
	logger.Info().Msg("Downgrade 2.8 -> 2.7 started")

	logger.Info().Msg("Rename keys was started")
	if err := downgradeSubscriptionKeyForAnonymous(logger, dataBase); err != nil {
		return err
	}

	if err := downgradeContactKeyForAnonymous(logger, dataBase); err != nil {
		return err
	}

	logger.Info().Msg("Downgrade 2.8 -> 2.7 was finished")
	return nil
}

var (
	subscriptionKeyForAnonymousOld = "moira-user-subscriptions:"
	subscriptionKeyForAnonymousNew = "moira-user-subscriptions:anonymous"
	contactKeyForAnonymousOld      = "moira-user-contacts:"
	contactKeyForAnonymousNew      = "moira-user-contacts:anonymous"
)

func updateSubscriptionKeyForAnonymous(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, subscriptionKeyForAnonymousOld, subscriptionKeyForAnonymousNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("updateSubscriptionKeyForAnonymous done")

	return nil
}

func updateContactKeyForAnonymous(logger moira.Logger, database moira.Database) error {
	err := renameKey(database, contactKeyForAnonymousOld, contactKeyForAnonymousNew)
	if err != nil {
		return err
	}

	logger.Info().Msg("updateContactKeyForAnonymous done")

	return nil
}

func downgradeSubscriptionKeyForAnonymous(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, subscriptionKeyForAnonymousNew, subscriptionKeyForAnonymousOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("downgradeSubscriptionKeyForAnonymous done")

	return nil
}

func downgradeContactKeyForAnonymous(logger moira.Logger, database moira.Database) error {
	err := changeKeysPrefix(database, contactKeyForAnonymousNew, contactKeyForAnonymousOld)
	if err != nil {
		return err
	}

	logger.Info().Msg("downgradeContactKeyForAnonymous done")

	return nil
}
