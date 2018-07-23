// nolint
package dto

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/expression"
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
	// Determines if trigger should alert when value is >= (true) or <= (false) threshold, By default we assume, IsRising = true
	IsRising *bool `json:"is_rising,omitempty"`
	// Set of triggers to manipulate subscriptions
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
}

// ToMoiraTrigger transforms TriggerModel to moira.Trigger
func (model *TriggerModel) ToMoiraTrigger() *moira.Trigger {
	return &moira.Trigger{
		ID:         model.ID,
		Name:       model.Name,
		Desc:       model.Desc,
		Targets:    model.Targets,
		WarnValue:  model.WarnValue,
		ErrorValue: model.ErrorValue,
		IsRising:   model.IsRising,
		Tags:       model.Tags,
		TTLState:   model.TTLState,
		TTL:        model.TTL,
		Schedule:   model.Schedule,
		Expression: &model.Expression,
		Patterns:   model.Patterns,
	}
}

// CreateTriggerModel transforms moira.Trigger to TriggerModel
func CreateTriggerModel(trigger *moira.Trigger) TriggerModel {
	return TriggerModel{
		ID:         trigger.ID,
		Name:       trigger.Name,
		Desc:       trigger.Desc,
		Targets:    trigger.Targets,
		WarnValue:  trigger.WarnValue,
		ErrorValue: trigger.ErrorValue,
		IsRising:   trigger.IsRising,
		Tags:       trigger.Tags,
		TTLState:   trigger.TTLState,
		TTL:        trigger.TTL,
		Schedule:   trigger.Schedule,
		Expression: moira.UseString(trigger.Expression),
		Patterns:   trigger.Patterns,
	}
}

func (trigger *Trigger) Bind(request *http.Request) error {
	if len(trigger.Targets) == 0 {
		return fmt.Errorf("targets is required")
	}
	if len(trigger.Tags) == 0 {
		return fmt.Errorf("tags is required")
	}
	reservedTagsFound := checkTriggerTags(trigger.Tags)
	if len(reservedTagsFound) > 0 {
		forbiddenTags := strings.Join(reservedTagsFound, ", ")
		return fmt.Errorf("forbidden tags: %s", forbiddenTags)
	}
	if trigger.Name == "" {
		return fmt.Errorf("trigger name is required")
	}
	if trigger.WarnValue == nil && trigger.ErrorValue == nil && trigger.Expression == "" {
		return fmt.Errorf("at least one of error_value, warn_value or expression is required")
	}
	if trigger.IsRising == nil {
		flag := true
		trigger.IsRising = &flag
	}
	if err := checkWarnErrorValues(trigger.WarnValue, trigger.ErrorValue, trigger.IsRising); err != nil {
		return err
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
		IsRising:                trigger.IsRising,
		PreviousState:           checker.NODATA,
		Expression:              &trigger.Expression,
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

	for _, tar := range trigger.Targets {
		database := middleware.GetDatabase(request)
		result, err := target.EvaluateTarget(database, tar, now-600, now, false)
		if err != nil {
			return err
		}

		trigger.Patterns = append(trigger.Patterns, result.Patterns...)

		if targetNum == 1 {
			expressionValues.MainTargetValue = 42
			for _, timeSeries := range result.TimeSeries {
				timeSeriesNames[timeSeries.Name] = true
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

func checkTriggerTags(tags []string) []string {
	reservedTagsFound := make([]string, 0)
	for _, tag := range tags {
		switch tag {
		case moira.EventHighDegradationTag, moira.EventDegradationTag, moira.EventProgressTag:
			reservedTagsFound = append(reservedTagsFound, tag)
		}
	}
	return reservedTagsFound
}

func checkWarnErrorValues(warn, error *float64, isRising *bool) error {
	if warn != nil && error != nil {
		if *warn == *error {
			return fmt.Errorf("error_value is equal to warn_value, please set exactly one value")
		}
		if *isRising && *warn > *error {
			return fmt.Errorf("error_value should be greater than warn_value")
		}
		if !*isRising && *warn < *error {
			return fmt.Errorf("warn_value should be greater than error_value")
		}
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
