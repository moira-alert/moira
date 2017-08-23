package reply

import (
	"github.com/moira-alert/moira-alert"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
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
