package expression

import (
	"fmt"
	"strings"

	"github.com/patrickmn/go-cache"

	"github.com/Knetic/govaluate"
	"github.com/moira-alert/moira"
)

var exprWarnErrorRising, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : (t1 >= WARN_VALUE ? WARN : OK)")
var exprWarnErrorFalling, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : (t1 <= WARN_VALUE ? WARN : OK)")
var exprWarnRising, _ = govaluate.NewEvaluableExpression("t1 >= WARN_VALUE ? WARN : OK")
var exprErrRising, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : OK")
var exprWarnFalling, _ = govaluate.NewEvaluableExpression("t1 <= WARN_VALUE ? WARN : OK")
var exprErrFalling, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : OK")

var exprCache = cache.New(cache.NoExpiration, cache.NoExpiration)

// ErrInvalidExpression represents bad expression or its state error
type ErrInvalidExpression struct {
	internalError error
}

func (err ErrInvalidExpression) Error() string {
	return err.internalError.Error()
}

// TriggerExpression represents trigger expression handler parameters, what can be used for trigger expression handling
type TriggerExpression struct {
	Expression *string

	WarnValue   *float64
	ErrorValue  *float64
	TriggerType string

	MainTargetValue         float64
	AdditionalTargetsValues map[string]float64
	PreviousState           moira.State
}

// Get realizing govaluate.Parameters interface used in evaluable expression
func (triggerExpression TriggerExpression) Get(name string) (interface{}, error) {
	switch name {
	case "OK":
		return moira.StateOK, nil
	case "WARN", "WARNING":
		return moira.StateWARN, nil
	case "ERROR":
		return moira.StateERROR, nil
	case "NODATA":
		return moira.StateNODATA, nil
	case "WARN_VALUE":
		if triggerExpression.WarnValue == nil {
			return nil, fmt.Errorf("no value with name WARN_VALUE")
		}
		return *triggerExpression.WarnValue, nil
	case "ERROR_VALUE":
		if triggerExpression.ErrorValue == nil {
			return nil, fmt.Errorf("no value with name ERROR_VALUE")
		}
		return *triggerExpression.ErrorValue, nil
	case "t1":
		return triggerExpression.MainTargetValue, nil
	case "PREV_STATE":
		return triggerExpression.PreviousState, nil
	default:
		value, ok := triggerExpression.AdditionalTargetsValues[name]
		if !ok {
			return nil, fmt.Errorf("no value with name %s", name)
		}
		return value, nil
	}
}

// Evaluate gets trigger expression and evaluates it for given parameters using govaluate
func (triggerExpression *TriggerExpression) Evaluate() (moira.State, error) {
	expr, err := getExpression(triggerExpression)
	if err != nil {
		return "", ErrInvalidExpression{internalError: err}
	}
	result, err := expr.Eval(triggerExpression)
	if err != nil {
		return "", ErrInvalidExpression{internalError: err}
	}
	switch res := result.(type) {
	case moira.State:
		return res, nil
	default:
		return "", ErrInvalidExpression{internalError: fmt.Errorf("expression result must be state value")}
	}
}

func getExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.TriggerType == moira.ExpressionTrigger {
		if triggerExpression.Expression == nil || *triggerExpression.Expression == "" {
			return nil, fmt.Errorf("trigger_type set to expression, but no expression provided")
		}
		return getUserExpression(triggerExpression)
	}
	return getSimpleExpression(triggerExpression)
}

func getSimpleExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.ErrorValue == nil && triggerExpression.WarnValue == nil {
		return nil, fmt.Errorf("error value and warning value can not be empty")
	}
	switch triggerExpression.TriggerType {
	case "":
		return nil, fmt.Errorf("trigger_type is not set")
	case moira.FallingTrigger:
		if triggerExpression.ErrorValue != nil && triggerExpression.WarnValue != nil {
			return exprWarnErrorFalling, nil
		} else if triggerExpression.ErrorValue != nil {
			return exprErrFalling, nil
		} else {
			return exprWarnFalling, nil
		}
	case moira.RisingTrigger:
		if triggerExpression.ErrorValue != nil && triggerExpression.WarnValue != nil {
			return exprWarnErrorRising, nil
		} else if triggerExpression.ErrorValue != nil {
			return exprErrRising, nil
		} else {
			return exprWarnRising, nil
		}
	}
	return nil, fmt.Errorf("wrong set of parametres: warn_value - %v, error_value - %v, trigger_type: %v",
		triggerExpression.WarnValue, triggerExpression.ErrorValue, triggerExpression.TriggerType)
}

func getUserExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if expr, found := exprCache.Get(*triggerExpression.Expression); found {
		return expr.(*govaluate.EvaluableExpression), nil
	}

	expr, err := govaluate.NewEvaluableExpression(*triggerExpression.Expression)

	if err != nil {
		if strings.Contains(err.Error(), "Undefined function") {
			return nil, fmt.Errorf("functions is forbidden")
		}
		return nil, err
	}
	if err := triggerExpression.validateUserExpression(expr.Vars()); err != nil {
		return nil, err
	}
	exprCache.Add(*triggerExpression.Expression, expr, cache.NoExpiration)
	return expr, nil
}

// This expression validation catches those errors which are ignored by govaluate
func (triggerExpression *TriggerExpression) validateUserExpression(vars []string) error {
	for _, v := range vars {
		if _, err := triggerExpression.Get(v); err != nil {
			return err
		}
	}
	if isTernaryExpression(*triggerExpression.Expression) && !checkforColon(*triggerExpression.Expression) || !checkForEmptyStates(*triggerExpression.Expression) {
		return fmt.Errorf("Invalid syntax")
	}
	return nil
}

func checkForEmptyStates(expression string) bool {
	anyReturnValueExist := false
	for i, c := range expression {
		if c == ':' {
			return checkForEmptyStates(expression[i+1 : len(expression)])
		}
		if c != ' ' && c != '(' && c != ')' {
			anyReturnValueExist = true
		}
	}
	return anyReturnValueExist
}

func checkforColon(expression string) bool {
	for _, c := range expression {
		if c == ':' {
			return true
		}
	}
	return false
}

func isTernaryExpression(expression string) bool {
	for _, c := range expression {
		if c == '?' {
			return true
		}
	}
	return false
}
