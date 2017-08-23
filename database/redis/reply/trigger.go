package reply

import (
	"github.com/moira-alert/moira-alert"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

func Trigger(rep interface{}, err error) (*moira.TriggerData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	trigger := &moira.TriggerData{}
	err = json.Unmarshal(bytes, trigger)
	if err != nil {
		return nil, err
	}
	return trigger, nil
}
