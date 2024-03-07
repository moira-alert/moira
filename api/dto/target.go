package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-graphite/carbonapi/expr/functions"
	"github.com/go-graphite/carbonapi/expr/metadata"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
)

func init() {
	functions.New(nil)
}

type typeOfProblem string

const (
	isWarn typeOfProblem = "warn"
	isBad  typeOfProblem = "bad"
)

var (
	unstableFunctions = map[string]bool{
		"removeAbovePercentile": true,
		"removeBelowPercentile": true,
		"smartSummarize":        true,
		"summarize":             true,
	}

	visualFunctions = map[string]bool{
		"alpha":           true,
		"areaBetween":     true,
		"color":           true,
		"consolidateBy":   true,
		"cumulative":      true,
		"dashed":          true,
		"drawAsInfinite":  true,
		"lineWidth":       true,
		"secondYAxis":     true,
		"setXFilesFactor": true,
		"sortBy":          true,
		"sortByMaxima":    true,
		"sortByMinima":    true,
		"sortByName":      true,
		"sortByTotal":     true,
		"stacked":         true,
		"threshold":       true,
		"verticalLine":    true,
	}

	falseNotificationsFunctions = map[string]bool{
		"averageAbove":             true,
		"averageBelow":             true,
		"averageOutsidePercentile": true,
		"currentAbove":             true,
		"currentBelow":             true,
		"filterSeries":             true,
		"highest":                  true,
		"highestAverage":           true,
		"highestCurrent":           true,
		"highestMax":               true,
		"limit":                    true,
		"lowest":                   true,
		"lowestAverage":            true,
		"lowestCurrent":            true,
		"maximumAbove":             true,
		"maximumBelow":             true,
		"minimumAbove":             true,
		"minimumBelow":             true,
		"mostDeviant":              true,
		"removeBetweenPercentile":  true,
		"removeEmptySeries":        true,
		"useSeriesAbove":           true,
	}

	timedFunctions = map[string]bool{
		"delay":                    true,
		"divideSeries":             true,
		"exponentialMovingAverage": true,
		"integralByInterval":       true,
		"linearRegression":         true,
		"movingAverage":            true,
		"movingMax":                true,
		"movingMedian":             true,
		"movingMin":                true,
		"movingSum":                true,
		"movingWindow":             true,
		"randomWalk":               true,
		"randomWalkFunction":       true,
		"sin":                      true,
		"sinFunction":              true,
		"summarize":                true,
		"time":                     true,
		"timeFunction":             true,
		"timeShift":                true,
		"timeSlice":                true,
		"timeStack":                true,
	}
)

type ProblemOfTarget struct {
	Argument    string            `json:"argument" example:"consolidateBy"`
	Type        typeOfProblem     `json:"type,omitempty" example:"warn"`
	Description string            `json:"description,omitempty" example:"This function affects only visual graph representation. It is meaningless in Moira"`
	Position    int               `json:"position" example:"0"`
	Problems    []ProblemOfTarget `json:"problems,omitempty"`
}

func (p *ProblemOfTarget) hasError() bool {
	if p.Type == isBad {
		return true
	}

	for _, pp := range p.Problems {
		if pp.hasError() {
			return true
		}
	}

	return false
}

type TreeOfProblems struct {
	SyntaxOk       bool             `json:"syntax_ok" example:"true"`
	TreeOfProblems *ProblemOfTarget `json:"tree_of_problems,omitempty" extensions:"x-nullable"`
}

// TargetVerification validates trigger targets.
func TargetVerification(targets []string, ttl time.Duration, triggerSource moira.TriggerSource) ([]TreeOfProblems, error) {
	switch triggerSource {
	case moira.PrometheusRemote:
		return []TreeOfProblems{{SyntaxOk: true}}, nil

	case moira.GraphiteLocal, moira.GraphiteRemote:
		return graphiteTargetVerification(targets, ttl, triggerSource), nil
	}

	return nil, fmt.Errorf("unknown trigger source '%s'", triggerSource)
}

func graphiteTargetVerification(targets []string, ttl time.Duration, triggerSource moira.TriggerSource) []TreeOfProblems {
	functionsOfTargets := make([]TreeOfProblems, 0)

	for _, target := range targets {
		functionsOfTarget := TreeOfProblems{SyntaxOk: true}

		expr, nestedExpr, err := parser.ParseExpr(target)
		if err != nil {
			functionsOfTarget.SyntaxOk = false
			functionsOfTargets = append(functionsOfTargets, functionsOfTarget)
			continue
		}
		isSpaceInMetricName := nestedExpr != ""
		if isSpaceInMetricName {
			functionsOfTarget.SyntaxOk = false
			functionsOfTargets = append(functionsOfTargets, functionsOfTarget)
			continue
		}

		functionsOfTarget.TreeOfProblems = checkExpression(expr, ttl, triggerSource)
		functionsOfTargets = append(functionsOfTargets, functionsOfTarget)
	}

	return functionsOfTargets
}

// DoesAnyTreeHaveError checks that at least one node of tree has a problem with error type.
// It is wrapper to handle slice of trees.
func DoesAnyTreeHaveError(trees []TreeOfProblems) bool {
	for _, tree := range trees {
		if tree.TreeOfProblems.hasError() {
			return true
		}
	}

	return false
}

// checkExpression validates expression.
func checkExpression(expression parser.Expr, ttl time.Duration, triggerSource moira.TriggerSource) *ProblemOfTarget {
	if !expression.IsFunc() && !strings.HasPrefix(expression.Target(), "seriesByTag") {
		return nil
	}

	funcName := expression.Target()
	problemFunction := checkFunction(funcName, triggerSource)

	if argument, ok := functionArgumentsInTheRangeTTL(expression, ttl); !ok {
		if problemFunction == nil {
			problemFunction = &ProblemOfTarget{Argument: funcName}
		}

		problemFunction.Problems = append(problemFunction.Problems, ProblemOfTarget{
			Argument: argument,
			Type:     isBad,
			Position: 1,
			Description: fmt.Sprintf(
				"The function %s has a time sampling parameter %s larger than allowed by the config:%s",
				funcName, expression.Args()[1].StringValue(), ttl.String()),
		})
	}

	for position, argument := range expression.Args() {
		if !argument.IsFunc() && !strings.HasPrefix(argument.Target(), "seriesByTag") {
			continue
		}

		if badFunc := checkExpression(argument, ttl, triggerSource); badFunc != nil {
			badFunc.Position = position

			if problemFunction == nil {
				problemFunction = &ProblemOfTarget{Argument: funcName}
			}

			problemFunction.Problems = append(problemFunction.Problems, *badFunc)
		}
	}

	return problemFunction
}

func checkFunction(funcName string, triggerSource moira.TriggerSource) *ProblemOfTarget {
	if triggerSource != moira.GraphiteLocal {
		return nil
	}

	if strings.HasPrefix(funcName, "seriesByTag") {
		valid, err := validateSeriesByTag(funcName)
		if err != nil {
			return &ProblemOfTarget{
				Argument:    funcName,
				Type:        isBad,
				Description: "Incorrect seriesByTag syntax.",
			}
		}
		if !valid {
			return &ProblemOfTarget{
				Argument:    funcName,
				Type:        isBad,
				Description: "seriesByTag function requires at least one argument with strict equality.",
			}
		}
		funcName = "seriesByTag"
	}

	if _, isUnstable := unstableFunctions[funcName]; isUnstable {
		return &ProblemOfTarget{
			Argument:    funcName,
			Type:        isBad,
			Description: "This function is unstable: it can return different historical values with each evaluation. Moira will show unexpected values that you don't see on your graphs.",
		}
	}

	if _, isFalseNotification := falseNotificationsFunctions[funcName]; isFalseNotification {
		return &ProblemOfTarget{
			Argument:    funcName,
			Type:        isWarn,
			Description: "This function shows and hides entire metric series based on their values. Moira will send frequent false NODATA notifications.",
		}
	}

	if _, isVisual := visualFunctions[funcName]; isVisual {
		return &ProblemOfTarget{
			Argument:    funcName,
			Type:        isWarn,
			Description: "This function affects only visual graph representation. It is meaningless in Moira.",
		}
	}

	if !funcIsSupported(funcName) {
		return &ProblemOfTarget{
			Argument:    funcName,
			Type:        isBad,
			Description: "Function is not supported, if you want to use it, switch to remote",
			Position:    0,
			Problems:    nil,
		}
	}

	return nil
}

// functionArgumentsInTheRangeTTL: Checking function arguments that they are in the range of TTL.
func functionArgumentsInTheRangeTTL(expression parser.Expr, ttl time.Duration) (string, bool) {
	if _, ok := timedFunctions[expression.Target()]; ok && len(expression.Args()) > 1 {
		argument, argumentDuration := positiveDuration(expression.Args()[1])
		return argument, argumentDuration <= ttl || ttl == 0
	}

	return "", true
}

func funcIsSupported(funcName string) bool {
	_, ok := metadata.FunctionMD.Functions[funcName]
	return ok || funcName == ""
}

// checks if a seriesByTag expression has at least one argument with a strict equality.
func validateSeriesByTag(target string) (bool, error) {
	tagArgs, err := filter.ParseSeriesByTag(target)

	if err != nil {
		return false, err
	}

	for _, arg := range tagArgs {
		if arg.Operator == filter.EqualOperator && !hasWildcard(arg.Value) {
			return true, nil
		}
	}

	return false, nil
}

func hasWildcard(target string) bool {
	return strings.ContainsAny(target, "[]{}*?")
}

// positiveDuration:
func positiveDuration(argument parser.Expr) (string, time.Duration) {
	var secondTimeDuration time.Duration
	var value string

	switch argument.Type() {
	case parser.EtConst:
		if secondArg := argument.FloatValue(); secondArg != 0 {
			value = fmt.Sprint(secondArg)
			secondTimeDuration = time.Duration(secondArg) * time.Second
		}
	case parser.EtString:
		value = argument.StringValue()
		second, _ := parser.IntervalString(value, 1)

		secondTimeDuration = time.Second * time.Duration(second)
	default: // 0 = EtName, 1 = EtFunc
	}

	if secondTimeDuration < 0 {
		secondTimeDuration *= -1
	}

	return value, secondTimeDuration
}
