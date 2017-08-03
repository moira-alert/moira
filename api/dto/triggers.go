package dto

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
	"net/http"
	"strings"
)

type TriggersList struct {
	Page  *int64                `json:"page,omitempty"`
	Size  *int64                `json:"size,omitempty"`
	Total *int64                `json:"total,omitempty"`
	List  []moira.TriggerChecks `json:"list"`
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
	val := float64(1000)
	expressionValues := map[string]*float64{
		"warn_value":  trigger.WarnValue,
		"error_value": trigger.ErrorValue,
		"PREV_STATE":  &val,
	}
	if err := resolvePatterns(request, trigger, expressionValues); err != nil {
		fmt.Printf("Invalid graphite targets %s: %s\n", trigger.Targets, err.Error())
		return fmt.Errorf("Invalid graphite targets")
	}
	if err := getExpression(trigger); err != nil {
		fmt.Printf("Invalid expression %s: %s\n", trigger.Expression, err.Error()) //todo
		return fmt.Errorf("Invalid expression")
	}
	return nil
}

func resolvePatterns(request *http.Request, trigger *Trigger, expressionValues map[string]*float64) error {
	isSimpleTrigger := true
	if len(trigger.Targets) > 1 {
		isSimpleTrigger = false
	}
	targetNum := 1
	timeSeriesNames := make([]string, 0)
	triggerPatterns := make(map[string]bool)

	for _, target := range trigger.Targets {
		expr2, _, err := expr.ParseExpr(target)
		if err != nil {
			return nil
		}
		patterns := expr2.Metrics()
		if isSimpleTrigger && !isSimpleTarget(patterns) {
			isSimpleTrigger = false
		}
		targetName := fmt.Sprintf("t%v", targetNum)
		for _, pattern := range patterns {
			database := request.Context().Value("database").(moira.Database)
			metrics, err := database.GetPatternMetrics(pattern.Metric)
			if err != nil {
				return err
			}
			timeSeriesNames = append(timeSeriesNames, metrics...)
			triggerPatterns[pattern.Metric] = true
		}
		val := float64(42)
		expressionValues[targetName] = &val
		targetNum += 1
	}

	trigger.Patterns = make([]string, 0, len(triggerPatterns))
	trigger.IsSimpleTrigger = isSimpleTrigger
	for key, _ := range triggerPatterns {
		trigger.Patterns = append(trigger.Patterns, key)
	}
	return nil
}

func isSimpleTarget(metrics []expr.MetricRequest) bool {
	if len(metrics) > 1 {
		return false
	}

	for _, metric := range metrics {
		if strings.ContainsAny(metric.Metric, "*{") {
			return false
		}
	}
	return true
}

func getExpression(trigger *Trigger) error {
	//todo Функция, которая преобразует WarnValue, ErrorValue и Expression в функцию питона для графита
	return nil
}

func (*Trigger) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type TriggerCheck struct {
	*moira.CheckData
	TriggerId string `json:"trigger_id"`
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
	Id      string `json:"id"`
	Message string `json:"message"`
}

func (*SaveTriggerResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
