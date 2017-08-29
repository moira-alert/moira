package reply

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
)

func Event(rep interface{}, err error) (moira.NotificationEvent, error) {
	event := moira.NotificationEvent{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return event, database.ErrNil
		}
		return event, err
	}
	err = json.Unmarshal(bytes, &event)
	if err != nil {
		return event, err
	}
	return event, nil
}

func Events(rep interface{}, err error) ([]*moira.NotificationEvent, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
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
