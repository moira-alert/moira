package controller

import (
	"regexp"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
)

const (
	zeroPage      int64 = 0
	allEventsSize int64 = -1
)

// GetTriggerEvents gets trigger events from current page and total count of filtered trigger events. Events list is filtered by time range
// with `from` and `to` params (`from` and `to` should be "+inf", "-inf" or int64 converted to string),
// by metric (regular expression) and by states. If `states` map is empty or nil then all states are accepted.
func GetTriggerEvents(
	database moira.Database,
	triggerID string,
	page, size int64,
	from, to string,
	metricRegexp *regexp.Regexp,
	states map[string]struct{},
) (*dto.EventsList, *api.ErrorResponse) {
	events, err := getFilteredNotificationEvents(database, triggerID, from, to, metricRegexp, states)
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	eventCount := int64(len(events))

	if page >= 0 {
		if size >= 0 {
			start := page * size
			end := start + size

			if start >= eventCount {
				events = []*moira.NotificationEvent{}
			} else {
				if end > eventCount {
					end = eventCount
				}

				events = events[start:end]
			}
		}

		if page > 0 && size < 0 {
			events = []*moira.NotificationEvent{}
		}

		// if page == 0 and size < 0 return all events
	} else {
		events = []*moira.NotificationEvent{}
	}

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
	from, to string,
	metricRegexp *regexp.Regexp,
	states map[string]struct{},
) ([]*moira.NotificationEvent, error) {
	events, err := database.GetNotificationEvents(triggerID, zeroPage, allEventsSize, from, to)
	if err != nil {
		return nil, err
	}

	return filterNotificationEvents(events, metricRegexp, states), nil
}

func filterNotificationEvents(
	notificationEvents []*moira.NotificationEvent,
	metricRegexp *regexp.Regexp,
	states map[string]struct{},
) []*moira.NotificationEvent {
	filteredNotificationEvents := make([]*moira.NotificationEvent, 0)

	for _, event := range notificationEvents {
		if metricRegexp.MatchString(event.Metric) {
			_, ok := states[string(event.State)]
			if len(states) == 0 || ok {
				filteredNotificationEvents = append(filteredNotificationEvents, event)
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
