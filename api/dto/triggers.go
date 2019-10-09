// nolint
package dto

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/expression"
	metricSource "github.com/moira-alert/moira/metric_source"
)

var targetNameRegex = regexp.MustCompile("t(\\d+)")

type TriggersList struct {
	Page  *int64               `json:"page,omitempty"`
	Size  *int64               `json:"size,omitempty"`
	Total *int64               `json:"total,omitempty"`
	Pager *string              `json:"pager,omitempty"`
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
	TTLState *moira.TTLState `json:"ttl_state,omitempty"`
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
	// If true, first event NODATA → OK will be omitted
	MuteNewMetrics bool `json:"mute_new_metrics"`
	// A list of targets that have only alone metrics
	AloneMetrics map[string]bool `json:"alone_metrics"`
}

// ToMoiraTrigger transforms TriggerModel to moira.Trigger
func (model *TriggerModel) ToMoiraTrigger() *moira.Trigger {
	return &moira.Trigger{
		ID:             model.ID,
		Name:           model.Name,
		Desc:           model.Desc,
		Targets:        model.Targets,
		WarnValue:      model.WarnValue,
		ErrorValue:     model.ErrorValue,
		TriggerType:    model.TriggerType,
		Tags:           model.Tags,
		TTLState:       model.TTLState,
		TTL:            model.TTL,
		Schedule:       model.Schedule,
		Expression:     &model.Expression,
		Patterns:       model.Patterns,
		IsRemote:       model.IsRemote,
		MuteNewMetrics: model.MuteNewMetrics,
		AloneMetrics:   model.AloneMetrics,
	}
}

// CreateTriggerModel transforms moira.Trigger to TriggerModel
func CreateTriggerModel(trigger *moira.Trigger) TriggerModel {
	return TriggerModel{
		ID:             trigger.ID,
		Name:           trigger.Name,
		Desc:           trigger.Desc,
		Targets:        trigger.Targets,
		WarnValue:      trigger.WarnValue,
		ErrorValue:     trigger.ErrorValue,
		TriggerType:    trigger.TriggerType,
		Tags:           trigger.Tags,
		TTLState:       trigger.TTLState,
		TTL:            trigger.TTL,
		Schedule:       trigger.Schedule,
		Expression:     moira.UseString(trigger.Expression),
		Patterns:       trigger.Patterns,
		IsRemote:       trigger.IsRemote,
		MuteNewMetrics: trigger.MuteNewMetrics,
	}
}

func (trigger *Trigger) Bind(request *http.Request) error {
	trigger.Tags = normalizeTags(trigger.Tags)
	if len(trigger.Targets) == 0 {
		return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("targets is required")}
	}
	if len(trigger.Tags) == 0 {
		return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("tags is required")}
	}
	if trigger.Name == "" {
		return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("trigger name is required")}
	}
	if err := checkWarnErrorExpression(trigger); err != nil {
		return api.ErrInvalidRequestContent{ValidationError: err}
	}
	for targetName := range trigger.AloneMetrics {
		if !targetNameRegex.MatchString(targetName) {
			return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target name should be in pattern: t\\d+")}
		}
		targetIndexStr := targetNameRegex.FindStringSubmatch(targetName)[0]
		targetIndex, err := strconv.Atoi(targetIndexStr)
		if err != nil {
			return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target index should be valid number: %w", err)}
		}
		if targetIndex < 0 || targetIndex > len(trigger.Targets) {
			return api.ErrInvalidRequestContent{ValidationError: fmt.Errorf("alone metrics target index should be in range from 1 to length of targets")}
		}
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
		TriggerType:             trigger.TriggerType,
		PreviousState:           moira.StateNODATA,
		Expression:              &trigger.Expression,
	}

	metricsSourceProvider := middleware.GetTriggerTargetsSourceProvider(request)
	metricsSource, err := metricsSourceProvider.GetMetricSource(trigger.IsRemote)
	if err != nil {
		return err
	}

	if err := checkTTLSanity(trigger, metricsSource); err != nil {
		return api.ErrInvalidRequestContent{ValidationError: err}
	}

	if err := resolvePatterns(request, trigger, &triggerExpression, metricsSource); err != nil {
		return err
	}
	if _, err := triggerExpression.Evaluate(); err != nil {
		return err
	}

	return nil
}

func checkTTLSanity(trigger *Trigger, metricsSource metricSource.MetricSource) error {
	maximumAllowedTTL := metricsSource.GetMetricsTTLSeconds()

	if trigger.TTL > maximumAllowedTTL {
		triggerType := "local"
		if trigger.IsRemote {
			triggerType = "remote"
		}
		return fmt.Errorf("TTL for %s trigger can't be more than %d seconds", triggerType, maximumAllowedTTL)
	}
	return nil
}

func resolvePatterns(request *http.Request, trigger *Trigger, expressionValues *expression.TriggerExpression, metricsSource metricSource.MetricSource) error {
	now := time.Now().Unix()
	targetNum := 1
	trigger.Patterns = make([]string, 0)
	metricsDataNames := make(map[string]bool)

	for _, tar := range trigger.Targets {
		fetchResult, err := metricsSource.Fetch(tar, now-600, now, false)
		if err != nil {
			return err
		}
		targetPatterns, err := fetchResult.GetPatterns()
		if err == nil {
			trigger.Patterns = append(trigger.Patterns, targetPatterns...)
		}

		if targetNum == 1 {
			expressionValues.MainTargetValue = 42
			for _, metricData := range fetchResult.GetMetricsData() {
				metricsDataNames[metricData.Name] = true
			}
		} else {
			targetName := fmt.Sprintf("t%v", targetNum)
			expressionValues.AdditionalTargetsValues[targetName] = 42
		}
		targetNum++
	}
	middleware.SetTimeSeriesNames(request, metricsDataNames)
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
		if err := checkSimpleModeFields(trigger); err != nil {
			return err
		}

	case moira.FallingTrigger:
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			if *trigger.WarnValue < *trigger.ErrorValue {
				return fmt.Errorf("warn_value should be greater than error_value")
			}
		}
		if err := checkSimpleModeFields(trigger); err != nil {
			return err
		}

	case moira.ExpressionTrigger:
		if trigger.Expression == "" {
			return fmt.Errorf("trigger_type set to expression, but no expression provided")
		}
		if trigger.WarnValue != nil && trigger.ErrorValue != nil {
			return fmt.Errorf("can't use 'warn_value' and 'error_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}
		if trigger.WarnValue != nil {
			return fmt.Errorf("can't use 'warn_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}
		if trigger.ErrorValue != nil {
			return fmt.Errorf("can't use 'error_value' on trigger_type: '%v'", moira.ExpressionTrigger)
		}

	default:
		return fmt.Errorf("wrong trigger_type: %v, allowable values: '%v', '%v', '%v'",
			trigger.TriggerType, moira.RisingTrigger, moira.FallingTrigger, moira.ExpressionTrigger)
	}

	return nil
}

func checkSimpleModeFields(trigger *Trigger) error {
	if len(trigger.Targets) > 1 {
		return fmt.Errorf("can't use trigger_type not '%v' for with multiple targets", trigger.TriggerType)
	}
	if trigger.Expression != "" {
		return fmt.Errorf("can't use 'expression' to trigger_type: '%v'", trigger.TriggerType)
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

type TriggerMaintenance struct {
	Trigger *int64           `json:"trigger"`
	Metrics map[string]int64 `json:"metrics"`
}

func (*TriggerMaintenance) Bind(r *http.Request) error {
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

type TriggerMetrics map[string]map[string][]moira.MetricValue

func (*TriggerMetrics) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
