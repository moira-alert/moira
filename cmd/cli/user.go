package main

import (
	"github.com/moira-alert/moira"
)

func transferUserSubscriptionsAndContacts(database moira.Database) error {
	if *fromUser == "" && *toUser == "" {
		return nil
	}

	contactIDs, err := database.GetUserContactIDs(*fromUser)
	if err != nil {
		return err
	}
	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return err
	}
	for _, contact := range contacts {
		contact.User = *toUser
		if err = database.SaveContact(contact); err != nil {
			return err
		}
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(*userDel)
	if err != nil {
		return err
	}
	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		subscription.User = *toUser
	}
	return database.SaveSubscriptions(subscriptions)
}

func deleteUser(database moira.Database) error {
	if *userDel == "" {
		return nil
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(*userDel)
	if err != nil {
		return err
	}
	for _, subscriptionID := range subscriptionIDs {
		if err = database.RemoveSubscription(subscriptionID); err != nil {
			return err
		}
	}

	contactIDs, err := database.GetUserContactIDs(*userDel)
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
