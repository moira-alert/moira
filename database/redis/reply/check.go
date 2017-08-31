package reply

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
)

// Check converts redis DB reply to moira.CheckData
func Check(rep interface{}, err error) (moira.CheckData, error) {
	check := moira.CheckData{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return check, database.ErrNil
		}
		return check, fmt.Errorf("Failed to read lastCheck: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &check)
	if err != nil {
		return check, fmt.Errorf("Failed to parse lastCheck json %s: %s", string(bytes), err.Error())
	}
	return check, nil
}
