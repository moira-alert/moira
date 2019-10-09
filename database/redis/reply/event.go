package reply

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

type notificationEventStorageElement struct {
	IsTriggerEvent   bool               `json:"trigger_event,omitempty"`
	Timestamp        int64              `json:"timestamp"`
	Metric           string             `json:"metric"`
	Value            *float64           `json:"value,omitempty"`
	Values           map[string]float64 `json:"values,omitempty"`
	State            moira.State        `json:"state"`
	TriggerID        string             `json:"trigger_id"`
	SubscriptionID   *string            `json:"sub_id,omitempty"`
	ContactID        string             `json:"contactId,omitempty"`
	OldState         moira.State        `json:"old_state"`
	Message          *string            `json:"msg,omitempty"`
	MessageEventInfo *moira.EventInfo   `json:"event_message"`
}

func toNotificationEventStorageElement(event moira.NotificationEvent) notificationEventStorageElement {
	//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
	if event.Value == nil {
		if value, ok := event.Values[firstTarget]; ok {
			event.Value = &value
		}
	}
	//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0
	return notificationEventStorageElement{
		IsTriggerEvent:   event.IsTriggerEvent,
		Timestamp:        event.Timestamp,
		Metric:           event.Metric,
		Value:            event.Value,
		Values:           event.Values,
		State:            event.State,
		TriggerID:        event.TriggerID,
		SubscriptionID:   event.SubscriptionID,
		ContactID:        event.ContactID,
		OldState:         event.OldState,
		Message:          event.Message,
		MessageEventInfo: event.MessageEventInfo,
	}
}

func (e notificationEventStorageElement) toNotificationEvent() moira.NotificationEvent {
	//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
	if e.Values == nil {
		e.Values = make(map[string]float64)
	}
	if e.Value != nil {
		e.Values[firstTarget] = *e.Value
		e.Value = nil
	}
	//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0
	return moira.NotificationEvent{
		IsTriggerEvent:   e.IsTriggerEvent,
		Timestamp:        e.Timestamp,
		Metric:           e.Metric,
		Value:            e.Value,
		Values:           e.Values,
		State:            e.State,
		TriggerID:        e.TriggerID,
		SubscriptionID:   e.SubscriptionID,
		ContactID:        e.ContactID,
		OldState:         e.OldState,
		Message:          e.Message,
		MessageEventInfo: e.MessageEventInfo,
	}
}

// GetEventBytes is a function that takes moira.NotificationEvent and turns it to bytes that will be saved in redis.
func GetEventBytes(event moira.NotificationEvent) ([]byte, error) {
	eventSE := toNotificationEventStorageElement(event)
	bytes, err := json.Marshal(eventSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %s", err.Error())
	}
	return bytes, nil
}

// Event converts redis DB reply to moira.NotificationEvent object
func Event(rep interface{}, err error) (moira.NotificationEvent, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return moira.NotificationEvent{}, database.ErrNil
		}
		return moira.NotificationEvent{}, fmt.Errorf("failed to read event: %s", err.Error())
	}
	eventSE := notificationEventStorageElement{}
	err = json.Unmarshal(bytes, &eventSE)
	if err != nil {
		return moira.NotificationEvent{}, fmt.Errorf("failed to parse event json %s: %s", string(bytes), err.Error())
	}
	return eventSE.toNotificationEvent(), nil
}

// Events converts redis DB reply to moira.NotificationEvent objects array
func Events(rep interface{}, err error) ([]*moira.NotificationEvent, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.NotificationEvent, 0), nil
		}
		return nil, fmt.Errorf("failed to read events: %s", err.Error())
	}
	events := make([]*moira.NotificationEvent, len(values))
	for i, value := range values {
		event, err2 := Event(value, err)
		if err2 != nil && err2 != database.ErrNil {
			return nil, err2
		} else if err2 == database.ErrNil {
			events[i] = nil
		} else {
			events[i] = &event
		}
	}
	return events, nil
}
