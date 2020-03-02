package reply

import (
	"encoding/json"
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/internal/database"
)

// Check converts redis DB reply to moira.CheckData
func Check(rep interface{}, err error) (moira2.CheckData, error) {
	check := moira2.CheckData{}
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
