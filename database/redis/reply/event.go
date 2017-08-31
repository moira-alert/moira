package reply

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
)

// Event converts redis DB reply to moira.NotificationEvent object
func Event(rep interface{}, err error) (moira.NotificationEvent, error) {
	event := moira.NotificationEvent{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return event, database.ErrNil
		}
		return event, fmt.Errorf("Failed to read event: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &event)
	if err != nil {
		return event, fmt.Errorf("Failed to parse event json %s: %s", string(bytes), err.Error())
	}
	return event, nil
}

// Events converts redis DB reply to moira.NotificationEvent objects array
func Events(rep interface{}, err error) ([]*moira.NotificationEvent, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.NotificationEvent, 0), nil
		}
		return nil, fmt.Errorf("Failed to read events: %s", err.Error())
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
