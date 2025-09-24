package controller

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetContactEventsHistoryByID is a controller that fetches events from database by using moira.Database.GetNotificationsHistoryByContactID.
func GetContactEventsHistoryByID(database moira.Database, contactID string, from, to, page, size int64,
) (*dto.ContactEventItemList, *api.ErrorResponse) {
	events, err := database.GetNotificationsHistoryByContactID(contactID, from, to, page, size)
	if err != nil {
		return nil, api.ErrorInternalServer(fmt.Errorf("GetContactEventsHistoryByID: can't get notifications for contact with id %v", contactID))
	}

	total, err := database.GetNotificationsTotalByContactID(contactID, from, to)
	if err != nil {
		return nil, api.ErrorInternalServer(
			fmt.Errorf("GetContactEventsHistoryByID: can't get total notifications count for contact with id %v", contactID),
		)
	}

	eventsList := dto.ContactEventItemList{
		List:  make([]dto.ContactEventItem, 0, len(events)),
		Page:  page,
		Size:  size,
		Total: total,
	}

	for _, i := range events {
		contactEventItem := &dto.ContactEventItem{
			TimeStamp: i.TimeStamp,
			Metric:    i.Metric,
			State:     string(i.State),
			OldState:  string(i.OldState),
			TriggerID: i.TriggerID,
		}
		eventsList.List = append(eventsList.List, *contactEventItem)
	}

	return &eventsList, nil
}
