package reply

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
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
	TTLState         *moira.TTLState     `json:"ttl_state,omitempty"`
	Schedule         *moira.ScheduleData `json:"sched,omitempty"`
	Expression       *string             `json:"expr,omitempty"`
	PythonExpression *string             `json:"expression,omitempty"`
	Patterns         []string            `json:"patterns"`
	TTL              string              `json:"ttl,omitempty"`
	IsRemote         bool                `json:"is_remote"`
	TriggerSource    moira.TriggerSource `json:"trigger_source,omitempty"`
	MuteNewMetrics   bool                `json:"mute_new_metrics,omitempty"`
	AloneMetrics     map[string]bool     `json:"alone_metrics"`
	CreatedAt        *int64              `json:"created_at"`
	UpdatedAt        *int64              `json:"updated_at"`
	CreatedBy        string              `json:"created_by"`
	UpdatedBy        string              `json:"updated_by"`
}

func (storageElement *triggerStorageElement) toTrigger() moira.Trigger {
	//TODO(litleleprikon): START remove in moira v2.8.0. Compatibility with moira < v2.6.0
	if storageElement.AloneMetrics == nil {
		aloneMetricsLen := len(storageElement.Targets)
		storageElement.AloneMetrics = make(map[string]bool, aloneMetricsLen)
		for i := 2; i <= aloneMetricsLen; i++ {
			targetName := fmt.Sprintf("t%d", i)
			storageElement.AloneMetrics[targetName] = true
		}
	}
	//TODO(litleleprikon): END remove in moira v2.8.0. Compatibility with moira < v2.6.0

	triggerSource := storageElement.TriggerSource.FillInIfNotSet(storageElement.IsRemote)
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
		TriggerSource:    triggerSource,
		MuteNewMetrics:   storageElement.MuteNewMetrics,
		AloneMetrics:     storageElement.AloneMetrics,
		CreatedAt:        storageElement.CreatedAt,
		UpdatedAt:        storageElement.UpdatedAt,
		CreatedBy:        storageElement.CreatedBy,
		UpdatedBy:        storageElement.UpdatedBy,
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
		IsRemote:         trigger.TriggerSource == moira.GraphiteRemote,
		TriggerSource:    trigger.TriggerSource,
		MuteNewMetrics:   trigger.MuteNewMetrics,
		AloneMetrics:     trigger.AloneMetrics,
		CreatedAt:        trigger.CreatedAt,
		UpdatedAt:        trigger.UpdatedAt,
		CreatedBy:        trigger.CreatedBy,
		UpdatedBy:        trigger.UpdatedBy,
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
func Trigger(rep *redis.StringCmd) (moira.Trigger, error) {
	bytes, err := rep.Bytes()
	if err != nil {
		if err == redis.Nil {
			return moira.Trigger{}, database.ErrNil
		}
		return moira.Trigger{}, fmt.Errorf("failed to read trigger: %s", err.Error())
	}
	triggerSE := &triggerStorageElement{}
	err = json.Unmarshal(bytes, triggerSE)
	if err != nil {
		return moira.Trigger{}, fmt.Errorf("failed to parse trigger json %s: %s", string(bytes), err.Error())
	}

	trigger := triggerSE.toTrigger()
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
