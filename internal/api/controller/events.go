package controller

import (
	"github.com/moira-alert/moira/internal/api"
	"github.com/moira-alert/moira/internal/api/dto"
	moira2 "github.com/moira-alert/moira/internal/moira"
)

// GetTriggerEvents gets trigger event from current page and all trigger event count
func GetTriggerEvents(database moira2.Database, triggerID string, page int64, size int64) (*dto.EventsList, *api.ErrorResponse) {
	events, err := database.GetNotificationEvents(triggerID, page*size, size-1)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	eventCount := database.GetNotificationEventCount(triggerID, -1)

	eventsList := &dto.EventsList{
		Size:  size,
		Page:  page,
		Total: eventCount,
		List:  make([]moira2.NotificationEvent, 0),
	}
	for _, event := range events {
		if event != nil {
			eventsList.List = append(eventsList.List, *event)
		}
	}
	return eventsList, nil
}

// DeleteAllEvents deletes all notification events
func DeleteAllEvents(database moira2.Database) *api.ErrorResponse {
	if err := database.RemoveAllNotificationEvents(); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
