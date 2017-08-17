package checker

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"strings"
)

var default1, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : (t1 >= WARN_VALUE ? WARN : OK)")
var default2, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : (t1 <= WARN_VALUE ? WARN : OK)")

var cache map[string]*govaluate.EvaluableExpression = make(map[string]*govaluate.EvaluableExpression, 0)

type ExpressionValues struct {
	WarnValue  *float64
	ErrorValue *float64

	MainTargetValue         float64
	AdditionalTargetsValues map[string]float64
	PreviousState           string
}

func (values ExpressionValues) Get(name string) (interface{}, error) {
	switch name {
	case "OK":
		return OK, nil
	case "WARN":
		return WARN, nil
	case "WARNING":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	case "NODATA":
		return NODATA, nil
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

func EvaluateExpression(triggerExpression *string, expressionValues ExpressionValues) (string, error) {
	expression, err := getExpression(triggerExpression, expressionValues)
	if err != nil {
		return "", err
	}
	result, err := expression.Eval(expressionValues)
	if err != nil {
		return "", err
	}
	switch res := result.(type) {
	case string:
		return res, nil
	default:
		return "", fmt.Errorf("Expression result must be state value")
	}
}

func getExpression(triggerExpression *string, values ExpressionValues) (*govaluate.EvaluableExpression, error) {
	if triggerExpression != nil && *triggerExpression != "" {
		return getUserExpression(*triggerExpression)
	} else {
		return getSimpleExpression(values)
	}
}

func getSimpleExpression(values ExpressionValues) (*govaluate.EvaluableExpression, error) {
	if values.ErrorValue == nil || values.WarnValue == nil {
		return nil, fmt.Errorf("Error value and Warning value can not be empty")
	}
	if *values.ErrorValue > *values.WarnValue {
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
	expression, err := govaluate.NewEvaluableExpression(triggerExpression)
	if err != nil {
		if strings.Contains(err.Error(), "Undefined function") {
			return nil, fmt.Errorf("Functions is forbidden")
		}
		return nil, err
	}
	cache[triggerExpression] = expression
	return expression, nil
}
