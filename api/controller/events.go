package controller

import (
	"regexp"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

// GetTriggerEvents gets trigger event from current page and all trigger event count. Events list is filtered by time range
// (`from` and `to` params), metric (regular expression) and states. If `states` map is empty or nil then all states are accepted.
func GetTriggerEvents(database moira.Database, triggerID string, page, size, from, to int64, metricRegexp *regexp.Regexp, states map[string]struct{},
) (*dto.EventsList, *api.ErrorResponse) {
	events, err := getFilteredNotificationEvents(database, triggerID, page, size, from, to, metricRegexp, states)
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

func getFilteredNotificationEvents(
	database moira.Database,
	triggerID string,
	page, size, from, to int64,
	metricRegexp *regexp.Regexp,
	states map[string]struct{},
) ([]*moira.NotificationEvent, error) {
	if size < 0 {
		events, err := database.GetNotificationEvents(triggerID, page, size, from, to)
		if err != nil {
			return nil, err
		}
		return filterNotificationEvents(events, metricRegexp, states), nil
	}

	filtered := make([]*moira.NotificationEvent, 0, size)
	var count int64

	for int64(len(filtered)) < size {
		eventsData, err := database.GetNotificationEvents(triggerID, page+count, size, from, to)
		if err != nil {
			return nil, err
		}

		if len(eventsData) == 0 {
			break
		}

		filtered = append(filtered, filterNotificationEvents(eventsData, metricRegexp, states)...)
		count += 1
	}

	return filtered, nil
}

func filterNotificationEvents(notificationEvents []*moira.NotificationEvent, metricRegexp *regexp.Regexp, states map[string]struct{}) []*moira.NotificationEvent {
	filteredNotificationEvents := make([]*moira.NotificationEvent, 0)

	for _, event := range notificationEvents {
		if metricRegexp.MatchString(event.Metric) {
			_, ok := states[string(event.State)]
			if len(states) == 0 || ok {
				filteredNotificationEvents = append(filteredNotificationEvents, event)
				continue
			}
		}
	}

	return filteredNotificationEvents
}

// DeleteAllEvents deletes all notification events.
func DeleteAllEvents(database moira.Database) *api.ErrorResponse {
	if err := database.RemoveAllNotificationEvents(); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
