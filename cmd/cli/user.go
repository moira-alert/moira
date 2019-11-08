package main

import (
	"bufio"
	"encoding/json"
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
		contact.User = to
		if err = database.SaveContact(contact); err != nil {
			return err
		}
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(from)
	if err != nil {
		return err
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return err
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

func handleCleanup(database moira.Database, conf cleanupConfig) error {
	var users []string

	reader := json.NewDecoder(bufio.NewReader(os.Stdin))

	if err := reader.Decode(&users); err != nil {
		return err
	}

	return usersCleanup(database, users, conf)
}

func usersCleanup(database moira.Database, users []string, config cleanupConfig) error {
	usersMap := make(map[string]bool, len(users)+len(config.Whitelist))

	if !config.HandlingAnonymous {
		config.Whitelist = append(config.Whitelist, "")
	}

	for _, user := range append(users, config.Whitelist...) {
		usersMap[user] = true
	}

	contacts, err := database.GetAllContacts()
	if err != nil {
		return nil
	}

	var usersNotFound []string

	for _, contact := range contacts {
		if !usersMap[contact.User] {
			usersNotFound = append(usersNotFound, contact.User)
		}
	}

	for _, user := range usersNotFound {
		if config.Delete {
			if err = deleteUser(database, user); err != nil {
				break
			}
		} else if err = offNotification(database, user); err != nil {
			break
		}
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

	for _, subscription := range subscriptions {
		subscription.Enabled = false
	}

	return database.SaveSubscriptions(subscriptions)
}
