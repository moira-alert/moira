package reply

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// NotifierState parses moira.NotifierState from redis.StringCmd.
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

// NotifierStateForSources represents map from metric source clusters to their states.
type NotifierStateForSources struct {
	States map[string]moira.NotifierState `json:"states"`
}

// ParseNotifierStateForSources parses NotifierStateBySources from redis.StringCmd.
func ParseNotifierStateForSources(rep *redis.StringCmd) (NotifierStateForSources, error) {
	state := NotifierStateForSources{}

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
