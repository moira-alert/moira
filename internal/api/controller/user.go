package controller

import (
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// GetUserSettings gets user contacts and subscriptions
func GetUserSettings(database moira2.Database, userLogin string) (*dto.UserSettings, *api.ErrorResponse) {
	userSettings := &dto.UserSettings{
		User:          dto.User{Login: userLogin},
		Contacts:      make([]moira2.ContactData, 0),
		Subscriptions: make([]moira2.SubscriptionData, 0),
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	subscriptions, err := database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	for _, subscription := range subscriptions {
		if subscription != nil {
			userSettings.Subscriptions = append(userSettings.Subscriptions, *subscription)
		}
	}
	contactIDs, err := database.GetUserContactIDs(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	for _, contact := range contacts {
		if contact != nil {
			userSettings.Contacts = append(userSettings.Contacts, *contact)
		}
	}
	return userSettings, nil
}
