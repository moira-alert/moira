package reply

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// Duty hack for moira.Trigger TTL int64 and stored trigger TTL string compatibility
type triggerStorageElement struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Desc             *string             `json:"desc,omitempty"`
	Targets          []string            `json:"targets"`
	WarnValue        *float64            `json:"warn_value"`
	ErrorValue       *float64            `json:"error_value"`
	TriggerType      string              `json:"trigger_type,omitempty"`
	Tags             []string            `json:"tags"`
	TTLState         *string             `json:"ttl_state,omitempty"`
	Schedule         *moira.ScheduleData `json:"sched,omitempty"`
	Expression       *string             `json:"expr,omitempty"`
	PythonExpression *string             `json:"expression,omitempty"`
	Patterns         []string            `json:"patterns"`
	TTL              string              `json:"ttl,omitempty"`
}

func (storageElement *triggerStorageElement) toTrigger() moira.Trigger {
	return moira.Trigger{
		ID:               storageElement.ID,
		Name:             storageElement.Name,
		Desc:             storageElement.Desc,
		Targets:          storageElement.Targets,
		WarnValue:        storageElement.WarnValue,
		ErrorValue:       storageElement.ErrorValue,
		TriggerType:      storageElement.TriggerType,
		Tags:             storageElement.Tags,
		TTLState:         storageElement.TTLState,
		Schedule:         storageElement.Schedule,
		Expression:       storageElement.Expression,
		PythonExpression: storageElement.PythonExpression,
		Patterns:         storageElement.Patterns,
		TTL:              getTriggerTTL(storageElement.TTL),
	}
}

func toTriggerStorageElement(trigger *moira.Trigger, triggerID string) *triggerStorageElement {
	return &triggerStorageElement{
		ID:               triggerID,
		Name:             trigger.Name,
		Desc:             trigger.Desc,
		Targets:          trigger.Targets,
		WarnValue:        trigger.WarnValue,
		ErrorValue:       trigger.ErrorValue,
		TriggerType:      trigger.TriggerType,
		Tags:             trigger.Tags,
		TTLState:         trigger.TTLState,
		Schedule:         trigger.Schedule,
		Expression:       trigger.Expression,
		PythonExpression: trigger.PythonExpression,
		Patterns:         trigger.Patterns,
		TTL:              getTriggerTTLString(trigger.TTL),
	}
}

func getTriggerTTL(ttlString string) int64 {
	if ttlString == "" {
		return 0
	}
	ttl, _ := strconv.ParseInt(ttlString, 10, 64)
	return ttl
}

func getTriggerTTLString(ttl int64) string {
	return fmt.Sprintf("%v", ttl)
}

// Trigger converts redis DB reply to moira.Trigger object
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

	trigger := triggerSE.toTrigger()
	convertTriggerIfNecessary(&trigger)

	return trigger, nil
}

// GetTriggerBytes marshal moira.Trigger to bytes array
func GetTriggerBytes(triggerID string, trigger *moira.Trigger) ([]byte, error) {
	triggerSE := toTriggerStorageElement(trigger, triggerID)
	bytes, err := json.Marshal(triggerSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trigger: %s", err.Error())
	}
	return bytes, nil
}

// convertTriggerIfNecessary converts moira.Trigger to a new format implemented in Moira 2.3 release
// Difference: in Moira 2.3 trigger could have 3 possible fields:
//  - WarnValue - warning threshold
//  - ErrorValue - error threshold
//  - Expression - custom govaluate expression
// In Moira 2.3 there is another field - TriggerType, it can take one of the following options:
//  - rising: error > warning > ok
//  - falling: error < warning < ok
//  - expression: trigger has custom expression
func convertTriggerIfNecessary(trigger *moira.Trigger) {
	switch trigger.TriggerType {
	case moira.RisingTrigger, moira.FallingTrigger, moira.ExpressionTrigger:
		return
	}
	setProperTriggerType(trigger)
}

func setProperTriggerType(trigger *moira.Trigger) {
	if trigger.Expression != nil && *trigger.Expression != "" {
		trigger.TriggerType = moira.ExpressionTrigger
		return
	}

	trigger.TriggerType = moira.RisingTrigger

	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		if *trigger.ErrorValue < *trigger.WarnValue {
			trigger.TriggerType = moira.FallingTrigger
			return
		}
		if *trigger.ErrorValue == *trigger.WarnValue {
			trigger.WarnValue = nil
			return
		}
	}
}
