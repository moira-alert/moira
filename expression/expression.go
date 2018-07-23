package expression

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Knetic/govaluate"
)

var defaultWarnErrorRising, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : (t1 >= WARN_VALUE ? WARN : OK)")
var defaultWarnErrorFalling, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : (t1 <= WARN_VALUE ? WARN : OK)")
var defaultWarnRising, _ = govaluate.NewEvaluableExpression("t1 >= WARN_VALUE ? WARN : OK")
var defaultErrRising, _ = govaluate.NewEvaluableExpression("t1 >= ERROR_VALUE ? ERROR : OK")
var defaultWarnFalling, _ = govaluate.NewEvaluableExpression("t1 <= WARN_VALUE ? WARN : OK")
var defaultErrFalling, _ = govaluate.NewEvaluableExpression("t1 <= ERROR_VALUE ? ERROR : OK")

var cache = make(map[string]*govaluate.EvaluableExpression)
var cacheLock sync.Mutex

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

	WarnValue  *float64
	ErrorValue *float64
	IsRising   *bool

	MainTargetValue         float64
	AdditionalTargetsValues map[string]float64
	PreviousState           string
}

// Get realizing govaluate.Parameters interface used in evaluable expression
func (triggerExpression TriggerExpression) Get(name string) (interface{}, error) {
	switch name {
	case "OK":
		return "OK", nil
	case "WARN", "WARNING":
		return "WARN", nil
	case "ERROR":
		return "ERROR", nil
	case "NODATA":
		return "NODATA", nil
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
		return "", ErrInvalidExpression{internalError: fmt.Errorf("expression result must be state value")}
	}
}

func getExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.Expression != nil && *triggerExpression.Expression != "" {
		return getUserExpression(*triggerExpression.Expression)
	}
	return getSimpleExpression(triggerExpression)
}

func getSimpleExpression(triggerExpression *TriggerExpression) (*govaluate.EvaluableExpression, error) {
	if triggerExpression.ErrorValue == nil && triggerExpression.WarnValue == nil {
		return nil, fmt.Errorf("error value and warning value can not be empty")
	}
	if triggerExpression.IsRising == nil {
		return nil, fmt.Errorf("choose if thresholds are rising or falling")
	}
	if triggerExpression.ErrorValue != nil && *triggerExpression.IsRising == true {
		return defaultErrRising, nil
	}
	if triggerExpression.ErrorValue != nil && *triggerExpression.IsRising == false {
		return defaultErrFalling, nil
	}
	if triggerExpression.WarnValue != nil && *triggerExpression.IsRising == true {
		return defaultWarnRising, nil
	}
	if triggerExpression.WarnValue != nil && *triggerExpression.IsRising == false {
		return defaultWarnFalling, nil
	}
	if *triggerExpression.ErrorValue >= *triggerExpression.WarnValue {
		return defaultWarnErrorRising, nil
	}
	return defaultWarnErrorFalling, nil
}

func getUserExpression(triggerExpression string) (*govaluate.EvaluableExpression, error) {
	err := evaluateAndCacheExpressionIfNeed(triggerExpression)
	if err != nil {
		return nil, err
	}
	return cache[triggerExpression], err
}

func evaluateAndCacheExpressionIfNeed(triggerExpression string) error {
	if _, ok := cache[triggerExpression]; !ok {
		cacheLock.Lock()
		defer cacheLock.Unlock()
		if _, ok := cache[triggerExpression]; !ok {
			newCache := make(map[string]*govaluate.EvaluableExpression, len(cache)+1)
			for k, v := range cache {
				newCache[k] = v
			}
			expr, err := govaluate.NewEvaluableExpression(triggerExpression)
			if err != nil {
				if strings.Contains(err.Error(), "Undefined function") {
					return fmt.Errorf("functions is forbidden")
				}
				return err
			}
			newCache[triggerExpression] = expr
			cache = newCache
		}
	}
	return nil
}
