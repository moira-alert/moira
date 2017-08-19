package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

//GetTriggerEvents gets trigger event from current page and all trigger event count
func GetTriggerEvents(database moira.Database, triggerID string, page int64, size int64) (*dto.EventsList, *api.ErrorResponse) {
	events, err := database.GetEvents(triggerID, page*size, size-1)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	eventCount := database.GetTriggerEventsCount(triggerID, -1)

	eventsList := &dto.EventsList{
		Size:  size,
		Page:  page,
		Total: eventCount,
		List:  events,
	}

	return eventsList, nil
}
