package reply

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

func Event(rep interface{}, err error) (moira.EventData, error) {
	event := moira.EventData{}
	if err != nil {
		return event, err
	}
	switch rep := rep.(type) {
	case []byte:
		err = json.Unmarshal(rep, &event)
		return event, err
	case nil:
		return event, redis.ErrNil
	case redis.Error:
		return event, rep
	}
	return event, fmt.Errorf("reply: unexpected type for moira.EventData, got type %T", rep)
}

func Events(rep interface{}, err error) ([]*moira.EventData, error) {
	if err != nil {
		return nil, err
	}
	switch rep := rep.(type) {
	case []interface{}:
		events := make([]*moira.EventData, len(rep))
		for i := range rep {
			if rep[i] == nil {
				continue
			}
			err = json.Unmarshal(rep[i].([]byte), events[i])
			if err != nil {
				return nil, fmt.Errorf("reply: unexpected element type for []moira.EventData, got type %T", rep[i])
			}
		}
		return events, nil
	case nil:
		return nil, redis.ErrNil
	case redis.Error:
		return nil, rep
	}
	return nil, fmt.Errorf("reply: unexpected type for []moira.EventData, got type %T", rep)
}
