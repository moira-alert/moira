package reply

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

func Event(rep interface{}, err error) (*moira.NotificationEvent, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	event := &moira.NotificationEvent{}
	err = json.Unmarshal(bytes, event)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func Events(rep interface{}, err error) ([]*moira.NotificationEvent, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
	}
	events := make([]*moira.NotificationEvent, len(values))
	for i, kk := range values {
		event, err2 := Event(kk, err)
		if err2 != nil {
			return nil, err2
		}
		events[i] = event
	}
	return events, nil
}
