package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetUserSettings gets user contacts and subscriptions.
func GetUserSettings(database moira.Database, userLogin string, auth *api.Authorization) (*dto.UserSettings, *api.ErrorResponse) {
	userSettings := &dto.UserSettings{
		User: dto.User{
			Login:       userLogin,
			AuthEnabled: auth.IsEnabled(),
			Role:        auth.GetRole(userLogin),
		},
		Contacts:      make([]dto.Contact, 0),
		Subscriptions: make([]dto.Subscription, 0),
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
			userSettings.Subscriptions = append(userSettings.Subscriptions, dto.Subscription(*subscription))
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

	contactsScores, err := database.GetContactsScore(contactIDs)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	for _, contact := range contacts {
		if contact != nil {
			if contactScore := contactsScores[contact.ID]; contactScore != nil {
				contactDto := dto.NewContact(*contact, *contactScore)
				userSettings.Contacts = append(userSettings.Contacts, contactDto)
			}
		}
	}

	return userSettings, nil
}
