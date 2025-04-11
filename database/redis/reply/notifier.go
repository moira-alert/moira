package reply

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func NotifierState(rep *redis.StringCmd) (moira.NotifierState, error) {
	state := moira.NotifierState{}

	bytes, err := rep.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return state, database.ErrNil
		}

		return state, fmt.Errorf("failed to read state: %s", err.Error())
	}

	err = json.Unmarshal(bytes, &state)
	if err != nil {
		return state, fmt.Errorf("failed to parse state json %s %s", string(bytes), err.Error())
	}

	return state, nil
}
