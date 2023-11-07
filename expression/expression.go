package expression

import (
	"fmt"
	"strings"

	"github.com/antonmedv/expr"
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
	name = strings.ToLower(name)

	switch name {
	case "ok":
		return moira.StateOK, nil
	case "warn", "warning":
		return moira.StateWARN, nil
	case "error":
		return moira.StateERROR, nil
	case "nodata":
		return moira.StateNODATA, nil
	case "warn_value":
		if triggerExpression.WarnValue == nil {
			return nil, fmt.Errorf("no value with name WARN_VALUE")
		}
		return *triggerExpression.WarnValue, nil
	case "error_value":
		if triggerExpression.ErrorValue == nil {
			return nil, fmt.Errorf("no value with name ERROR_VALUE")
		}
		return *triggerExpression.ErrorValue, nil
	case "t1":
		return triggerExpression.MainTargetValue, nil
	case "prev_state":
		return triggerExpression.PreviousState, nil
	default:
		value, ok := triggerExpression.AdditionalTargetsValues[name]
		if !ok {
			return nil, fmt.Errorf("no value with name %s", name)
		}
		return value, nil
	}
}

// Validate returns error if triggers of type moira.ExpressionTrigger are badly formatted otherwise nil
func (triggerExpression *TriggerExpression) Validate() error {
	if triggerExpression.TriggerType != moira.ExpressionTrigger {
		return nil
	}
	if triggerExpression.Expression == nil || *triggerExpression.Expression == "" {
		return ErrInvalidExpression{
			internalError: fmt.Errorf("trigger_type set to expression, but no expression provided"),
		}
	}
	expression := *triggerExpression.Expression
	env := map[string]interface{}{
		"ok":         moira.StateOK,
		"error":      moira.StateERROR,
		"warn":       moira.StateWARN,
		"warning":    moira.StateWARN,
		"nodata":     moira.StateNODATA,
		"t1":         triggerExpression.MainTargetValue,
		"prev_state": triggerExpression.PreviousState,
	}
	if triggerExpression.WarnValue != nil {
		env["warn_value"] = *triggerExpression.WarnValue
	}
	if triggerExpression.ErrorValue != nil {
		env["error_value"] = *triggerExpression.ErrorValue
	}
	for k, v := range triggerExpression.AdditionalTargetsValues {
		env[k] = v
	}
	if _, err := expr.Compile(
		strings.ToLower(expression),
		expr.Optimize(true),
		expr.Env(env),
	); err != nil {
		return ErrInvalidExpression{err}
	}
	return nil
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
		return getUserExpression(*triggerExpression.Expression)
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

func getUserExpression(triggerExpression string) (*govaluate.EvaluableExpression, error) {
	if expr, found := exprCache.Get(triggerExpression); found {
		return expr.(*govaluate.EvaluableExpression), nil
	}

	expr, err := govaluate.NewEvaluableExpression(triggerExpression)
	if err != nil {
		if strings.Contains(err.Error(), "Undefined function") {
			return nil, fmt.Errorf("functions is forbidden")
		}
		return nil, err
	}

	exprCache.Add(triggerExpression, expr, cache.NoExpiration) //nolint
	return expr, nil
}
