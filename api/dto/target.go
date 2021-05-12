package dto

import (
	"fmt"
	"time"

	"github.com/go-graphite/carbonapi/expr/functions"
	"github.com/go-graphite/carbonapi/expr/metadata"

	"github.com/go-graphite/carbonapi/pkg/parser"
)

func init() {
	functions.New(nil)
}

type typeOfProblem string

const (
	isWarn typeOfProblem = "warn"
	isBad  typeOfProblem = "bad"
)

type checkTimedArgumentsFunc func(args parser.Expr) error

func checkTimedArgumentsSoFar(funcName string, ttl time.Duration, args ...parser.Expr) error {
	const argsCount = 2
	if l := len(args); l > argsCount {
		return fmt.Errorf("function %s should have only one timed argument, but have %d", funcName, l)
	}

	argument, argumentDuration := positiveDuration(args[1])
	if argumentDuration >= ttl && ttl != 0 {
		return argument
	}
}

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

	timedFunctions = map[string][]int{
		"delay":                    {1},
		"exponentialMovingAverage": {1},
		"integralByInterval":       {1},
		"linearRegression":         {1, 2},
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

type problemOfTarget struct {
	Argument    string            `json:"argument"`
	Type        typeOfProblem     `json:"type,omitempty"`
	Description string            `json:"description,omitempty"`
	Position    int               `json:"position"`
	Problems    []problemOfTarget `json:"problems,omitempty"`
}

type TreeOfProblems struct {
	SyntaxOk       bool             `json:"syntax_ok"`
	TreeOfProblems *problemOfTarget `json:"tree_of_problems,omitempty"`
}

func TargetVerification(targets []string, ttl time.Duration, isRemote bool) []TreeOfProblems {
	functionsOfTargets := make([]TreeOfProblems, 0)

	for _, target := range targets {
		functionsOfTarget := TreeOfProblems{SyntaxOk: true}

		expr, _, err := parser.ParseExpr(target)
		if err != nil {
			functionsOfTarget.SyntaxOk = false
			functionsOfTargets = append(functionsOfTargets, functionsOfTarget)
			continue
		}

		functionsOfTarget.TreeOfProblems = checkExpression(expr, ttl, isRemote)
		functionsOfTargets = append(functionsOfTargets, functionsOfTarget)
	}

	return functionsOfTargets
}

func checkExpression(expression parser.Expr, ttl time.Duration, isRemote bool) *problemOfTarget {
	switch {
	case expression.IsFunc():
		return checkFunction(expression, ttl, isRemote)
	case expression.IsName():
		// checkName
	default:
		return &problemOfTarget{Argument: expression.Target()}
	}
}

func checkFunction(expression parser.Expr, ttl time.Duration, isRemote bool) *problemOfTarget {
	funcName := expression.Target()

	if _, isUnstable := unstableFunctions[funcName]; isUnstable {
		return &problemOfTarget{
			Argument:    funcName,
			Type:        isBad,
			Description: fmt.Sprintf("Function %s is unstable: it can return different historical values with each evaluation. Moira will show unexpected values that you don't see on your graphs.", funcName),
		}
	}

	if _, isFalseNotification := falseNotificationsFunctions[funcName]; isFalseNotification {
		return &problemOfTarget{
			Argument:    funcName,
			Type:        isWarn,
			Description: fmt.Sprintf("Function %s shows and hides entire metric series based on their values. Moira will send frequent false NODATA notifications.", funcName),
		}
	}

	if _, isVisual := visualFunctions[funcName]; isVisual {
		return &problemOfTarget{
			Argument:    funcName,
			Type:        isWarn,
			Description: fmt.Sprintf("Function %s affects only visual graph representation. It is meaningless in Moira.", funcName),
		}
	}

	if !isRemote && !funcIsSupported(funcName) {
		return &problemOfTarget{
			Argument:    funcName,
			Type:        isBad,
			Description: fmt.Sprintf("Function %s is not supported, if you want to use it, switch to remote", funcName),
			Position:    0,
			Problems:    nil,
		}
	}

	if argument, ok := functionArgumentsInTheRangeTTL(expression, ttl); !ok {
		if problemFunction == nil {
			problemFunction = &problemOfTarget{Argument: target}
		}

		problemFunction.Problems = append(problemFunction.Problems, problemOfTarget{
			Argument: argument,
			Type:     isBad,
			Position: 1,
			Description: fmt.Sprintf(
				"The function %s has a time sampling parameter %s larger than allowed by the config:%s",
				target, expression.Args()[1].StringValue(), ttl.String()),
		})
	}

	for position, argument := range expression.Args() {
		if badFunc := checkExpression(argument, ttl, isRemote); badFunc != nil {
			badFunc.Position = position

			if problemFunction == nil {
				problemFunction = &problemOfTarget{Argument: target}
			}

			problemFunction.Problems = append(problemFunction.Problems, *badFunc)
		}
	}

	return nil
}

// functionArgumentsInTheRangeTTL: Checking function arguments that they are in the range of TTL
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
