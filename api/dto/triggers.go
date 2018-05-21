// nolint
package dto

import (
	"fmt"
	"net/http"
	"time"
	"strings"

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
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Desc       *string             `json:"desc,omitempty"`
	Targets    []string            `json:"targets"`
	WarnValue  *float64            `json:"warn_value"`
	ErrorValue *float64            `json:"error_value"`
	Tags       []string            `json:"tags"`
	TTLState   *string             `json:"ttl_state,omitempty"`
	TTL        int64               `json:"ttl,omitempty"`
	Schedule   *moira.ScheduleData `json:"sched,omitempty"`
	Expression string              `json:"expression"`
	Patterns   []string            `json:"patterns"`
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
	if trigger.WarnValue == nil && trigger.Expression == "" {
		return fmt.Errorf("warn_value is required")
	}
	if trigger.ErrorValue == nil && trigger.Expression == "" {
		return fmt.Errorf("error_value is required")
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
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

type TriggerMetrics struct {
	Main       map[string][]*moira.MetricValue `json:"main"`
	Additional map[string][]*moira.MetricValue `json:"additional,omitempty"`
}

func (*TriggerMetrics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
