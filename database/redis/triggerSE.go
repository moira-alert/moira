package redis

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"strconv"
)

type triggerStorageElement struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Desc            *string             `json:"desc,omitempty"`
	Targets         []string            `json:"targets"`
	WarnValue       *float64            `json:"warn_value"`
	ErrorValue      *float64            `json:"error_value"`
	Tags            []string            `json:"tags"`
	TtlState        *string             `json:"ttl_state,omitempty"`
	Schedule        *moira.ScheduleData `json:"sched,omitempty"`
	Expression      *string             `json:"expression,omitempty"`
	Patterns        []string            `json:"patterns"`
	IsSimpleTrigger bool                `json:"is_simple_trigger"`
	Ttl             *string             `json:"ttl"`
}

func toTrigger(storageElement *triggerStorageElement, triggerId string) *moira.Trigger {
	return &moira.Trigger{
		ID:              triggerId,
		Name:            storageElement.Name,
		Desc:            storageElement.Desc,
		Targets:         storageElement.Targets,
		WarnValue:       storageElement.WarnValue,
		ErrorValue:      storageElement.ErrorValue,
		Tags:            storageElement.Tags,
		TtlState:        storageElement.TtlState,
		Schedule:        storageElement.Schedule,
		Expression:      storageElement.Expression,
		Patterns:        storageElement.Patterns,
		IsSimpleTrigger: storageElement.IsSimpleTrigger,
		Ttl:             getTriggerTtl(storageElement.Ttl),
	}
}

func toTriggerStorageElement(trigger *moira.Trigger, triggerId string) *triggerStorageElement {
	return &triggerStorageElement{
		ID:              triggerId,
		Name:            trigger.Name,
		Desc:            trigger.Desc,
		Targets:         trigger.Targets,
		WarnValue:       trigger.WarnValue,
		ErrorValue:      trigger.ErrorValue,
		Tags:            trigger.Tags,
		TtlState:        trigger.TtlState,
		Schedule:        trigger.Schedule,
		Expression:      trigger.Expression,
		Patterns:        trigger.Patterns,
		IsSimpleTrigger: trigger.IsSimpleTrigger,
		Ttl:             getTriggerTtlString(trigger.Ttl),
	}
}

func getTriggerTtl(ttlString *string) *int64 {
	if ttlString == nil {
		return nil
	}
	ttl, _ := strconv.ParseInt(*ttlString, 10, 64)
	return &ttl
}

func getTriggerTtlString(ttl *int64) *string {
	if ttl == nil {
		return nil
	}
	ttlString := fmt.Sprintf("%v", *ttl)
	return &ttlString
}
