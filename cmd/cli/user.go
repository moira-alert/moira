package main

import (
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
