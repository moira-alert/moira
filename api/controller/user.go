package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetUserSettings(database moira.Database, userLogin string) (*dto.UserSettings, *dto.ErrorResponse) {
	//todo не забыть пропихнуть user в каждый subscription
	userSettings := &dto.UserSettings{
		User:          dto.User{Login: userLogin},
		Contacts:      make([]moira.ContactData, 0),
		Subscriptions: make([]moira.SubscriptionData, 0),
	}
	subscriptionIds, err := database.GetUserSubscriptionIds(userLogin)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	for _, id := range subscriptionIds {
		subscription, err := database.GetSubscription(id)
		if err != nil {
			//todo is it right?
			return nil, dto.ErrorInternalServer(err)
		}
		userSettings.Subscriptions = append(userSettings.Subscriptions, subscription)
	}

	contactIds, err := database.GetUserContacts(userLogin)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	for _, id := range contactIds {
		contact, err := database.GetContact(id)
		if err != nil {
			//todo is it right?
			return nil, dto.ErrorInternalServer(err)
		}
		userSettings.Contacts = append(userSettings.Contacts, contact)
	}

	return userSettings, nil
}
