package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetEvents(database moira.Database, triggerId string, page int64, size int64) (*dto.EventsList, *api.ErrorResponse) {
	events, err := database.GetEvents(triggerId, page*size, size-1)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	eventCount := database.GetTriggerEventsCount(triggerId, -1)

	eventsList := &dto.EventsList{
		Size:  size,
		Page:  page,
		Total: eventCount,
		List:  events,
	}

	return eventsList, nil
}
