package reply

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

func Check(rep interface{}, err error) (*moira.CheckData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	check := &moira.CheckData{}
	err = json.Unmarshal(bytes, check)
	if err != nil {
		return nil, err
	}
	return check, nil
}
