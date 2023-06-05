package controller

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

func GetContactByIdWithEventsLimit(database moira.Database, contactID string, from uint64, to uint64) (*dto.ContactWithEvents, *api.ErrorResponse) {
	contact, err := database.GetContact(contactID)
	if err != nil {
		return nil, api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get contact with id " + contactID))
	}

	events, err := database.GetNotificationsByContactIdWithLimit(contactID, from, to)

	if err != nil {
		return nil, api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get notifications for contact with id " + contactID))
	}

	var eventsList []moira.NotificationEventHistoryItem
	for _, i := range events {
		eventsList = append(eventsList, *i)
	}

	contactToReturn := &dto.ContactWithEvents{
		ID:     contact.ID,
		User:   contact.User,
		TeamID: contact.Team,
		Type:   contact.Type,
		Value:  contact.Value,
		Events: eventsList,
	}

	return contactToReturn, nil
}
