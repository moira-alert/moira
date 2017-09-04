// nolint
package dto

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/middleware"
	"github.com/moira-alert/moira-alert/checker"
	"github.com/moira-alert/moira-alert/expression"
	"github.com/moira-alert/moira-alert/target"
	"net/http"
	"strings"
	"time"
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
	moira.Trigger
	Throttling int64 `json:"throttling"`
}

func (trigger *Trigger) Bind(request *http.Request) error {
	if len(trigger.Targets) == 0 {
		return fmt.Errorf("targets is required")
	}
	if trigger.WarnValue == nil && trigger.Expression == nil {
		return fmt.Errorf("warn_value is required")
	}
	if trigger.ErrorValue == nil && trigger.Expression == nil {
		return fmt.Errorf("error_value is required")
	}
	if trigger.TTL == nil {
		return fmt.Errorf("TTL is required")
	}

	triggerExpression := expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64),
		WarnValue:               trigger.WarnValue,
		ErrorValue:              trigger.ErrorValue,
		PreviousState:           checker.NODATA,
		Expression:              trigger.Expression,
	}

	logger := middleware.GetLoggerEntry(request)

	if err := resolvePatterns(request, trigger, &triggerExpression); err != nil {
		logger.Infof("Invalid graphite targets %s: %s\n", trigger.Targets, err.Error())
		return err
	}
	if _, err := triggerExpression.Evaluate(); err != nil {
		logger.Infof("Invalid expression %s: %s\n", moira.UseString(trigger.Expression), err.Error())
		return err
	}
	return nil
}

func resolvePatterns(request *http.Request, trigger *Trigger, expressionValues *expression.TriggerExpression) error {
	trigger.IsSimpleTrigger = true
	if len(trigger.Targets) > 1 {
		trigger.IsSimpleTrigger = false
	}
	now := time.Now().Unix()
	targetNum := 1
	trigger.Patterns = make([]string, 0)
	timeSeriesNames := make(map[string]bool)

	for _, tar := range trigger.Targets {
		database := middleware.GetDatabase(request)
		result, err := target.EvaluateTarget(database, tar, now-600, now, true)
		if err != nil {
			return err
		}
		trigger.Patterns = append(trigger.Patterns, result.Patterns...)
		if trigger.IsSimpleTrigger && !isSimpleTarget(result.Patterns) {
			trigger.IsSimpleTrigger = false
		}
		for _, timeSeries := range result.TimeSeries {
			timeSeriesNames[timeSeries.Name] = true
		}
		if targetNum == 1 {
			expressionValues.MainTargetValue = 42
		} else {
			targetName := fmt.Sprintf("t%v", targetNum)
			expressionValues.AdditionalTargetsValues[targetName] = 42
		}
		targetNum++
	}
	middleware.SetTimeSeriesNames(request, timeSeriesNames)
	return nil
}

func isSimpleTarget(patterns []string) bool {
	if len(patterns) > 1 {
		return false
	}

	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "*{") {
			return false
		}
	}
	return true
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
