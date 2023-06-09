package controller

import (
	"fmt"
	"strconv"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

func GetContactByIdWithEventsLimit(database moira.Database, contactID string, from string, to string) (*dto.ContactWithEvents, *api.ErrorResponse) {
	if from == "" || to == "" {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should specified"))
	}

	fromInt, fromErr := strconv.ParseInt(from, 10, 64)
	toInt, toErr := strconv.ParseInt(to, 10, 64)

	if fromErr != nil || toErr != nil {
		return nil, api.ErrorInvalidRequest(fmt.Errorf("'from' and 'to' query params should be positive numbers"))
	}

	contact, err := database.GetContact(contactID)

	if err != nil {
		return nil, api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get contact with id %v", contactID))
	}

	events, err := database.GetNotificationsByContactIdWithLimit(contactID, fromInt, toInt)

	if err != nil {
		return nil, api.ErrorInternalServer(fmt.Errorf("GetContactByIdWithEventsLimit: can't get notifications for contact with id %v", contactID))
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
