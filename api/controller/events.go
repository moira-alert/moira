package controller

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"regexp"
)

// GetTriggerEvents gets trigger event from current page and all trigger event count. Events list is filtered by time range
// (`from` and `to` params), metric (regular expression) and states. If `states` map is empty or nil then all states are accepted.
func GetTriggerEvents(database moira.Database, triggerID string, page, size, from, to int64, metricRegexp *regexp.Regexp, states map[string]struct{}) (*dto.EventsList, *api.ErrorResponse) {
	events, err := database.GetNotificationEvents(triggerID, page*size, size, from, to, metricRegexp, states)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}
	eventCount := database.GetNotificationEventCount(triggerID, -1)

	eventsList := &dto.EventsList{
		Size:  size,
		Page:  page,
		Total: eventCount,
		List:  make([]moira.NotificationEvent, 0, len(events)),
	}
	for _, event := range events {
		if event != nil {
			eventsList.List = append(eventsList.List, *event)
		}
	}
	return eventsList, nil
}

// DeleteAllEvents deletes all notification events.
func DeleteAllEvents(database moira.Database) *api.ErrorResponse {
	if err := database.RemoveAllNotificationEvents(); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
