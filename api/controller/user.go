package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

//GetUserSettings gets user contacts and subscriptions
func GetUserSettings(database moira.Database, userLogin string) (*dto.UserSettings, *api.ErrorResponse) {
	userSettings := &dto.UserSettings{
		User:          dto.User{Login: userLogin},
		Contacts:      make([]*moira.ContactData, 0),
		Subscriptions: make([]*moira.SubscriptionData, 0),
	}

	subscriptionIDs, err := database.GetUserSubscriptionIDs(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	userSettings.Subscriptions, err = database.GetSubscriptions(subscriptionIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	contactIDs, err := database.GetUserContactIDs(userLogin)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	contacts, err := database.GetContacts(contactIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	userSettings.Contacts = contacts
	return userSettings, nil
}
