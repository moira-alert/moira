package reply

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// Check converts redis DB reply to moira.CheckData
func Check(rep interface{}, err error) (moira.CheckData, error) {
	check := moira.CheckData{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return check, database.ErrNil
		}
		return check, fmt.Errorf("failed to read lastCheck: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &check)
	if err != nil {
		return check, fmt.Errorf("failed to parse lastCheck json %s: %s", string(bytes), err.Error())
	}
	return check, nil
}
