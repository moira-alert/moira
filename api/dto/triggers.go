// nolint
package dto

import (
	"fmt"
	"net/http"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/remote"
	"github.com/moira-alert/moira/target"
)

type TriggersList struct {
	Page  *int64               `json:"page,omitempty"`
	Size  *int64               `json:"size,omitempty"`
	Total *int64               `json:"total,omitempty"`
	List  []moira.TriggerCheck `json:"list"`
}

func (*TriggersList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Trigger struct {
	TriggerModel
	Throttling int64 `json:"throttling"`
}

// TriggerModel is moira.Trigger api representation
type TriggerModel struct {
	// Trigger unique ID
	ID string `json:"id"`
	// Trigger name
	Name string `json:"name"`
	// Description string
	Desc *string `json:"desc,omitempty"`
	// Graphite-like targets: t1, t2, ...
	Targets []string `json:"targets"`
	// WARN threshold
	WarnValue *float64 `json:"warn_value"`
	// ERROR threshold
	ErrorValue *float64 `json:"error_value"`
	// Could be: rising, falling, expression
	TriggerType string `json:"trigger_type"`
	// Set of tags to manipulate subscriptions
	Tags []string `json:"tags"`
	// When there are no metrics for trigger, Moira will switch metric to TTLState state after TTL seconds
	TTLState *string `json:"ttl_state,omitempty"`
	// When there are no metrics for trigger, Moira will switch metric to TTLState state after TTL seconds
	TTL int64 `json:"ttl,omitempty"`
	// Determines when Moira should monitor trigger
	Schedule *moira.ScheduleData `json:"sched,omitempty"`
	// Used if you need more complex logic than provided by WARN/ERROR values
	Expression string `json:"expression"`
	// Graphite patterns for trigger
	Patterns []string `json:"patterns"`
	// Shows if trigger is remote (graphite-backend) based or stored inside Moira-Redis DB
	IsRemote bool `json:"is_remote"`
}

// ToMoiraTrigger transforms TriggerModel to moira.Trigger
func (model *TriggerModel) ToMoiraTrigger() *moira.Trigger {
	return &moira.Trigger{
		ID:          model.ID,
		Name:        model.Name,
		Desc:        model.Desc,
		Targets:     model.Targets,
		WarnValue:   model.WarnValue,
		ErrorValue:  model.ErrorValue,
		TriggerType: model.TriggerType,
		Tags:        model.Tags,
		TTLState:    model.TTLState,
		TTL:         model.TTL,
		Schedule:    model.Schedule,
		Expression:  &model.Expression,
		Patterns:    model.Patterns,
		IsRemote:    model.IsRemote,
	}
}

// CreateTriggerModel transforms moira.Trigger to TriggerModel
func CreateTriggerModel(trigger *moira.Trigger) TriggerModel {
	return TriggerModel{
		ID:          trigger.ID,
		Name:        trigger.Name,
		Desc:        trigger.Desc,
		Targets:     trigger.Targets,
		WarnValue:   trigger.WarnValue,
		ErrorValue:  trigger.ErrorValue,
		TriggerType: trigger.TriggerType,
		Tags:        trigger.Tags,
		TTLState:    trigger.TTLState,
		TTL:         trigger.TTL,
		Schedule:    trigger.Schedule,
		Expression:  moira.UseString(trigger.Expression),
		Patterns:    trigger.Patterns,
		IsRemote:    trigger.IsRemote,
	}
}

func (trigger *Trigger) Bind(request *http.Request) error {
	if len(trigger.Targets) == 0 {
		return fmt.Errorf("targets is required")
	}
	if len(trigger.Tags) == 0 {
		return fmt.Errorf("tags is required")
	}
	if trigger.Name == "" {
		return fmt.Errorf("trigger name is required")
	}
	if err := checkWarnErrorExpression(trigger); err != nil {
		return err
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
		TriggerType:             trigger.TriggerType,
		PreviousState:           checker.NODATA,
		Expression:              &trigger.Expression,
	}

	remoteCfg := middleware.GetRemoteConfig(request)
	if trigger.IsRemote && !remoteCfg.IsEnabled() {
		return fmt.Errorf("remote graphite storage is not enabled")
	}

	if err := resolvePatterns(request, trigger, &triggerExpression); err != nil {
		return err
	}
	if _, err := triggerExpression.Evaluate(); err != nil {
		return err
	}
	return nil
}

func resolvePatterns(request *http.Request, trigger *Trigger, expressionValues *expression.TriggerExpression) error {
	now := time.Now().Unix()
	targetNum := 1
	trigger.Patterns = make([]string, 0)
	timeSeriesNames := make(map[string]bool)

	remoteCfg := middleware.GetRemoteConfig(request)
	database := middleware.GetDatabase(request)
	var err error

	for _, tar := range trigger.Targets {
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteCfg, tar, now-600, now, false)
			if err != nil {
				return err
			}
		} else {
			result, err := target.EvaluateTarget(database, tar, now-600, now, false)
			if err != nil {
				return err
			}
			trigger.Patterns = append(trigger.Patterns, result.Patterns...)
			timeSeries = result.TimeSeries
		}

		if targetNum == 1 {
			expressionValues.MainTargetValue = 42
			for _, ts := range timeSeries {
				timeSeriesNames[ts.Name] = true
			}
		} else {
			targetName := fmt.Sprintf("t%v", targetNum)
			expressionValues.AdditionalTargetsValues[targetName] = 42
		}
		targetNum++
	}
	middleware.SetTimeSeriesNames(request, timeSeriesNames)
	return nil
}

func checkWarnErrorExpression(trigger *Trigger) error {
	if trigger.WarnValue == nil && trigger.ErrorValue == nil && trigger.Expression == "" {
		return fmt.Errorf("at least one of error_value, warn_value or expression is required")
	}

	if trigger.WarnValue != nil && trigger.ErrorValue != nil && *trigger.WarnValue == *trigger.ErrorValue {
		return fmt.Errorf("error_value is equal to warn_value, please set exactly one value")
	}

	switch trigger.TriggerType {
	case "":
		if trigger.Expression != "" {
			trigger.TriggerType = moira.ExpressionTrigger
			return nil
		}
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue > *trigger.ErrorValue {
				trigger.TriggerType = moira.FallingTrigger
				return nil
			}
			if *trigger.WarnValue < *trigger.ErrorValue {
				trigger.TriggerType = moira.RisingTrigger
				return nil
			}
		}
		if trigger.WarnValue == nil {
			return fmt.Errorf("warn_value: is empty - please fill both values or choose trigger_type: rising, falling, expression")
		}
		if trigger.ErrorValue == nil {
			return fmt.Errorf("error_value: is empty - please fill both values or choose trigger_type: rising, falling, expression")
		}

	case moira.RisingTrigger:
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue > *trigger.ErrorValue {
				return fmt.Errorf("error_value should be greater than warn_value")
			}
		}
	case moira.FallingTrigger:
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue < *trigger.ErrorValue {
				return fmt.Errorf("warn_value should be greater than error_value")
			}
		}
	case moira.ExpressionTrigger:
		if trigger.Expression == "" {
			return fmt.Errorf("trigger_type set to expression, but no expression provided")
		}
	default:
		return fmt.Errorf("wrong trigger_type: %v, allowable values: '%v', '%v', '%v'",
			trigger.TriggerType, moira.RisingTrigger, moira.FallingTrigger, moira.ExpressionTrigger)
	}

	return nil
}

func (*Trigger) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TriggerCheck struct {
	*moira.CheckData
	TriggerID string `json:"trigger_id"`
}

func (*TriggerCheck) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type MetricsMaintenance map[string]int64

func (*MetricsMaintenance) Bind(r *http.Request) error {
	return nil
}

type ThrottlingResponse struct {
	Throttling int64 `json:"throttling"`
}

func (*ThrottlingResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type SaveTriggerResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func (*SaveTriggerResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TriggerMetrics map[string][]moira.MetricValue

func (*TriggerMetrics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
