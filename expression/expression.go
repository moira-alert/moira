package expression

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"strings"
)

var default1, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : (t1 >= WARN_VALUE ? WARN : OK)")
var default2, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : (t1 <= WARN_VALUE ? WARN : OK)")

var cache map[string]*govaluate.EvaluableExpression = make(map[string]*govaluate.EvaluableExpression, 0)

type ErrInvalidExpression struct {
	internalError error
}

func (err ErrInvalidExpression) Error() string {
	return fmt.Sprintf("Invalid expression: %s", err.internalError.Error())
}

type TriggerExpression struct {
	Expression *string

	WarnValue  *float64
	ErrorValue *float64

	MainTargetValue         float64
	AdditionalTargetsValues map[string]float64
	PreviousState           string
}

func (values TriggerExpression) Get(name string) (interface{}, error) {
	switch name {
	case "OK":
		return "OK", nil
	case "WARN":
		return "WARN", nil
	case "WARNING":
		return "WARN", nil
	case "ERROR":
		return "ERROR", nil
	case "NODATA":
		return "NODATA", nil
	case "WARN_VALUE":
		if values.WarnValue == nil {
			return nil, fmt.Errorf("No value with name WARN_VALUE")
		}
		return *values.WarnValue, nil
	case "ERROR_VALUE":
		if values.ErrorValue == nil {
			return nil, fmt.Errorf("No value with name ERROR_VALUE")
		}
		return *values.ErrorValue, nil
	case "t1":
		return values.MainTargetValue, nil
	case "PREV_STATE":
		return values.PreviousState, nil
	default:
		value, ok := values.AdditionalTargetsValues[name]
		if !ok {
			return nil, fmt.Errorf("No value with name %s", name)
		}
		return value, nil
	}
}

func (triggerExpression *TriggerExpression) Evaluate() (string, error) {
	expr, err := getExpression(triggerExpression)
	if err != nil {
		return "", ErrInvalidExpression{internalError: err}
	}
	result, err := expr.Eval(triggerExpression)
	if err != nil {
		return "", ErrInvalidExpression{internalError: err}
	}
	switch res := result.(type) {
	case string:
		return res, nil
	default:
		return "", ErrInvalidExpression{internalError: fmt.Errorf("Expression result must be state value")}
	}
}

func getExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.Expression != nil && *triggerExpression.Expression != "" {
		return getUserExpression(*triggerExpression.Expression)
	} else {
		return getSimpleExpression(triggerExpression)
	}
}

func getSimpleExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.ErrorValue == nil || triggerExpression.WarnValue == nil {
		return nil, fmt.Errorf("Error value and Warning value can not be empty")
	}
	if *triggerExpression.ErrorValue > *triggerExpression.WarnValue {
		return default1, nil
	} else {
		return default2, nil
	}
}

func getUserExpression(triggerExpression string) (*govaluate.EvaluableExpression, error) {
	cached, ok := cache[triggerExpression]
	if ok {
		return cached, nil
	}
	expr, err := govaluate.NewEvaluableExpression(triggerExpression)
	if err != nil {
		if strings.Contains(err.Error(), "Undefined function") {
			return nil, fmt.Errorf("Functions is forbidden")
		}
		return nil, err
	}
	cache[triggerExpression] = expr
	return expr, nil
}
