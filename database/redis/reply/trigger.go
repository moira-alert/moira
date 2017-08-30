package reply

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"strconv"
)

//Duty hack for moira.Trigger TTL int64 and stored trigger TTL string compatibility
type triggerStorageElement struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Desc            *string             `json:"desc,omitempty"`
	Targets         []string            `json:"targets"`
	WarnValue       *float64            `json:"warn_value"`
	ErrorValue      *float64            `json:"error_value"`
	Tags            []string            `json:"tags"`
	TTLState        *string             `json:"ttl_state,omitempty"`
	Schedule        *moira.ScheduleData `json:"sched,omitempty"`
	Expression      *string             `json:"expression,omitempty"`
	Patterns        []string            `json:"patterns"`
	IsSimpleTrigger bool                `json:"is_simple_trigger"`
	TTL             *string             `json:"ttl"`
}

func (storageElement *triggerStorageElement) toTrigger() moira.Trigger {
	return moira.Trigger{
		ID:              storageElement.ID,
		Name:            storageElement.Name,
		Desc:            storageElement.Desc,
		Targets:         storageElement.Targets,
		WarnValue:       storageElement.WarnValue,
		ErrorValue:      storageElement.ErrorValue,
		Tags:            storageElement.Tags,
		TTLState:        storageElement.TTLState,
		Schedule:        storageElement.Schedule,
		Expression:      storageElement.Expression,
		Patterns:        storageElement.Patterns,
		IsSimpleTrigger: storageElement.IsSimpleTrigger,
		TTL:             getTriggerTtl(storageElement.TTL),
	}
}

func getTriggerTtl(ttlString *string) *int64 {
	if ttlString == nil {
		return nil
	}
	ttl, _ := strconv.ParseInt(*ttlString, 10, 64)
	return &ttl
}

func Trigger(rep interface{}, err error) (moira.Trigger, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return moira.Trigger{}, database.ErrNil
		}
		return moira.Trigger{}, fmt.Errorf("Failed to read trigger: %s", err.Error())
	}
	triggerSE := &triggerStorageElement{}
	err = json.Unmarshal(bytes, triggerSE)
	if err != nil {
		return moira.Trigger{}, fmt.Errorf("Failed to parse trigger json %s: %s", string(bytes), err.Error())
	}

	return triggerSE.toTrigger(), nil
}
