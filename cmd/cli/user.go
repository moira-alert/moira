package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/moira-alert/moira"
)

func transferUserSubscriptionsAndContacts(database moira.Database, from, to string) error {
	contactIDs, err := database.GetUserContactIDs(from)
	if err != nil {
		return err
	}

	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return err
	}

	for _, contact := range contacts {
		if contact == nil {
			continue
		}

		contact.User = to
		if err = database.SaveContact(contact); err != nil {
			return err
		}
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(from)
	if err != nil {
		return err
	}

	subscriptionsTmp, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return err
	}

	subscriptions := make([]*moira.SubscriptionData, 0, len(subscriptionsTmp))
	for _, subscription := range subscriptionsTmp {
		if subscription != nil {
			subscriptions = append(subscriptions, subscription)
		}
	}

	for _, subscription := range subscriptions {
		subscription.User = to
	}

	return database.SaveSubscriptions(subscriptions)
}

func deleteUser(database moira.Database, user string) error {
	subscriptionIDs, err := database.GetUserSubscriptionIDs(user)
	if err != nil {
		return err
	}

	for _, subscriptionID := range subscriptionIDs {
		if err = database.RemoveSubscription(subscriptionID); err != nil {
			return err
		}
	}

	contactIDs, err := database.GetUserContactIDs(user)
	if err != nil {
		return err
	}

	for _, contactID := range contactIDs {
		if err := database.RemoveContact(contactID); err != nil {
			return err
		}
	}

	return nil
}

func handleCleanup(logger moira.Logger, database moira.Database, config cleanupConfig) error {
	var users []string

	reader := json.NewDecoder(os.Stdin)

	if err := reader.Decode(&users); err != nil {
		return err
	}

	return usersCleanup(logger, database, users, config)
}

func usersCleanup(logger moira.Logger, database moira.Database, users []string, config cleanupConfig) error {
	if config.AddAnonymousToWhitelist {
		config.Whitelist = append(config.Whitelist, "")
	}

	usersMapLength := len(users) + len(config.Whitelist)
	const usersMapMaxLength = 100000
	if usersMapLength > usersMapMaxLength {
		return errors.New("users count is too large")
	}
	usersMap := make(map[string]bool, usersMapLength)

	for _, user := range append(users, config.Whitelist...) {
		usersMap[user] = true
	}

	contacts, err := database.GetAllContacts()
	if err != nil {
		return err
	}

	if len(contacts) == 0 {
		return nil
	}

	usersNotFound := make(map[string]bool, len(contacts))

	for _, contact := range contacts {
		if contact == nil {
			continue
		}

		if !usersMap[contact.User] {
			usersNotFound[contact.User] = true
		}
	}

	for user := range usersNotFound {
		if config.Delete {
			if err = deleteUser(database, user); err != nil {
				return err
			}
		} else {
			if err = offNotification(database, user); err != nil {
				return err
			}
		}

		logger.Debugb().
			String("user", user).
			Msg("User cleaned")
	}

	return err
}

func offNotification(database moira.Database, user string) error {
	subscriptionIDs, err := database.GetUserSubscriptionIDs(user)
	if err != nil {
		return err
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return err
	}

	saveSubscriptions := make([]*moira.SubscriptionData, 0, len(subscriptions))

	for _, subscription := range subscriptions {
		if subscription == nil {
			continue
		}

		if !subscription.Enabled {
			continue
		}

		subscription.Enabled = false
		saveSubscriptions = append(saveSubscriptions, subscription)
	}

	return database.SaveSubscriptions(saveSubscriptions)
}
