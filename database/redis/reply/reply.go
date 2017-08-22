package reply

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

func Event(rep interface{}, err error) (*moira.EventData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	event := &moira.EventData{}
	err = json.Unmarshal(bytes, event)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func Events(rep interface{}, err error) ([]*moira.EventData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
	}
	events := make([]*moira.EventData, len(values))
	for i, kk := range values {
		event, err := Event(kk, err)
		if err != nil {
			return nil, err
		}
		events[i] = event
	}
	return events, nil
}
