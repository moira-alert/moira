package plotting

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const (
	//thresholdTestValueIncrement               = float64(10)
	thresholdNegativeTestRisingWarnValue      = float64(-100)
	thresholdNegativeTestRisingErrorValue     = float64(0)
	thresholdNegativeTestFallingWarnValue     = thresholdNegativeTestRisingErrorValue
	thresholdNegativeTestFallingErrorValue    = thresholdNegativeTestRisingWarnValue
	thresholdNonNegativeTestRisingWarnValue   = float64(100)
	thresholdNonNegativeTestRisingErrorValue  = float64(200)
	thresholdNonNegativeTestFallingWarnValue  = thresholdNonNegativeTestRisingErrorValue
	thresholdNonNegativeTestFallingErrorValue = thresholdNonNegativeTestRisingWarnValue
)

var (
	innerNegativeTestCaseLimits = plotLimits{
		lowest:  -100,
		highest: 100,
	}
	outerNegativeTestCaseLimits = plotLimits{
		lowest:  100,
		highest: 200,
	}
	innerNonNegativeTestCaseLimits = plotLimits{
		lowest:  0,
		highest: 200,
	}
	outerNonNegativeTestCaseLimits = plotLimits{
		lowest:  1000,
		highest: 2000,
	}
)

// thresholdTestCase is a single threshold test case
type thresholdTestCase struct {
	name        string
	triggerType string
	warnValue   interface{}
	errorValue  interface{}
	limits      plotLimits
	expected    []*threshold
}

func (testCase *thresholdTestCase) getCaseMessage() string {
	caseMessage := bytes.NewBuffer([]byte("Trying to generate thresholds for "))
	caseMessage.WriteString(fmt.Sprintf("%s trigger:\n", testCase.triggerType))
	caseMessage.WriteString(fmt.Sprintf("lowest limit: %f, highest limit: %f\n",
		testCase.limits.lowest, testCase.limits.highest))
	if testCase.warnValue != nil {
		caseMessage.WriteString(fmt.Sprintf("WARN value: %f ", testCase.warnValue))
	}
	if testCase.errorValue != nil {
		caseMessage.WriteString(fmt.Sprintf("ERROR value: %f", testCase.errorValue))
	}
	if testCase.warnValue != nil || testCase.errorValue != nil {
		caseMessage.WriteString("\n")
	}
	expectedThresholds := ""
	if len(testCase.expected) > 0 {
		for _, expectedItem := range testCase.expected {
			expectedThresholds += fmt.Sprintf("%s threshold (y): %f ",
				expectedItem.thresholdType, expectedItem.yCoordinate)
		}
	} else {
		expectedThresholds = "no thresholds required"
	}
	caseMessage.WriteString(expectedThresholds)
	return caseMessage.String()
}

// thresholdNegativeTestCases is a collection of negative threshold test cases
var thresholdNegativeTestCases = []thresholdTestCase{
	{
		name:        "Negative | RISING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNegativeTestRisingWarnValue,
		errorValue:  nil,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | RISING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNegativeTestRisingErrorValue,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | RISING | {limits.lowest ..limits.highest}, warn, error",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNegativeTestRisingWarnValue,
		errorValue:  thresholdNegativeTestRisingErrorValue,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | RISING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNegativeTestRisingWarnValue,
		errorValue:  nil,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				yCoordinate:   innerNegativeTestCaseLimits.highest - thresholdNegativeTestRisingWarnValue,
			},
		},
	},
	{
		name:        "Negative | RISING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNegativeTestRisingErrorValue,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   innerNegativeTestCaseLimits.highest - thresholdNegativeTestRisingErrorValue,
			},
		},
	},
	{
		name:        "Negative | RISING | {limits.lowest <= warn << error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNegativeTestRisingWarnValue,
		errorValue:  thresholdNegativeTestRisingErrorValue,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   innerNegativeTestCaseLimits.highest - thresholdNegativeTestRisingErrorValue,
			},
			{
				thresholdType: "WARN",
				yCoordinate:   innerNegativeTestCaseLimits.highest - thresholdNegativeTestRisingWarnValue,
			},
		},
	},
	//{
	//	name:        "Negative | RISING | {limits.lowest <= warn < error <= limits.highest}",
	//	triggerType: moira.RisingTrigger,
	//	warnValue:   thresholdNegativeTestRisingWarnValue,
	//	errorValue:  thresholdNegativeTestRisingWarnValue + thresholdTestValueIncrement,
	//	limits:      innerNegativeTestCaseLimits,
	//	expected: []*threshold{
	//		{
	//			thresholdType: "ERROR",
	//			yCoordinate:   innerNegativeTestCaseLimits.highest - (thresholdNegativeTestRisingWarnValue + thresholdTestValueIncrement),
	//		},
	//	},
	//},
	{
		name:        "Negative | FALLING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNegativeTestFallingErrorValue,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | FALLING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNegativeTestFallingWarnValue,
		errorValue:  nil,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | FALLING | {limits.lowest ..limits.highest}, error, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNegativeTestFallingWarnValue,
		errorValue:  thresholdNegativeTestFallingErrorValue,
		limits:      outerNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Negative | FALLING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNegativeTestFallingErrorValue,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   thresholdNegativeTestFallingErrorValue,
			},
		},
	},
	{
		name:        "Negative | FALLING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNegativeTestFallingWarnValue,
		errorValue:  nil,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				yCoordinate:   thresholdNegativeTestFallingWarnValue,
			},
		},
	},
	{
		name:        "Negative | FALLING | {limits.lowest <= error << warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNegativeTestFallingWarnValue,
		errorValue:  thresholdNegativeTestFallingErrorValue,
		limits:      innerNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   thresholdNegativeTestFallingErrorValue,
			},
			{
				thresholdType: "WARN",
				yCoordinate:   thresholdNegativeTestFallingWarnValue,
			},
		},
	},
	//{
	//	name:        "Negative | FALLING | {limits.lowest <= error < warn <= limits.highest}",
	//	triggerType: moira.FallingTrigger,
	//	warnValue:   thresholdNegativeTestFallingErrorValue + thresholdTestValueIncrement,
	//	errorValue:  thresholdNegativeTestFallingErrorValue,
	//	limits:      innerNegativeTestCaseLimits,
	//	expected: []*threshold{
	//		{
	//			thresholdType: "ERROR",
	//			yCoordinate:   thresholdNegativeTestFallingErrorValue,
	//		},
	//	},
	//},
}

// thresholdNonNegativeTestCases is a collection of non-negative threshold test cases
var thresholdNonNegativeTestCases = []thresholdTestCase{
	{
		name:        "Non-negative | RISING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNonNegativeTestRisingWarnValue,
		errorValue:  nil,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | RISING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNonNegativeTestRisingErrorValue,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | RISING | {limits.lowest ..limits.highest}, warn, error",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNonNegativeTestRisingWarnValue,
		errorValue:  thresholdNonNegativeTestRisingErrorValue,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | RISING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNonNegativeTestRisingWarnValue,
		errorValue:  nil,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				yCoordinate:   innerNonNegativeTestCaseLimits.highest - thresholdNonNegativeTestRisingWarnValue,
			},
		},
	},
	{
		name:        "Non-negative | RISING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNonNegativeTestRisingErrorValue,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   innerNonNegativeTestCaseLimits.highest - thresholdNonNegativeTestRisingErrorValue,
			},
		},
	},
	{
		name:        "Non-negative | RISING | {limits.lowest <= warn << error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdNonNegativeTestRisingWarnValue,
		errorValue:  thresholdNonNegativeTestRisingErrorValue,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   innerNonNegativeTestCaseLimits.highest - thresholdNonNegativeTestRisingErrorValue,
			},
			{
				thresholdType: "WARN",
				yCoordinate:   innerNonNegativeTestCaseLimits.highest - thresholdNonNegativeTestRisingWarnValue,
			},
		},
	},
	//{
	//	name:        "Non-negative | RISING | {limits.lowest <= warn < error <= limits.highest}",
	//	triggerType: moira.RisingTrigger,
	//	warnValue:   thresholdNonNegativeTestRisingWarnValue,
	//	errorValue:  thresholdNonNegativeTestRisingWarnValue + thresholdTestValueIncrement,
	//	limits:      innerNonNegativeTestCaseLimits,
	//	expected: []*threshold{
	//		{
	//			thresholdType: "ERROR",
	//			yCoordinate:   innerNonNegativeTestCaseLimits.highest - (thresholdNonNegativeTestRisingWarnValue + thresholdTestValueIncrement),
	//		},
	//	},
	//},
	{
		name:        "Non-negative | FALLING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNonNegativeTestFallingErrorValue,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | FALLING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNonNegativeTestFallingWarnValue,
		errorValue:  nil,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | FALLING | {limits.lowest ..limits.highest}, error, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNonNegativeTestFallingWarnValue,
		errorValue:  thresholdNonNegativeTestFallingErrorValue,
		limits:      outerNonNegativeTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "Non-negative | FALLING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdNonNegativeTestFallingErrorValue,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   thresholdNonNegativeTestFallingErrorValue,
			},
		},
	},
	{
		name:        "Non-negative | FALLING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNonNegativeTestFallingWarnValue,
		errorValue:  nil,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				yCoordinate:   thresholdNonNegativeTestFallingWarnValue,
			},
		},
	},
	{
		name:        "Non-negative | FALLING | {limits.lowest <= error << warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdNonNegativeTestFallingWarnValue,
		errorValue:  thresholdNonNegativeTestFallingErrorValue,
		limits:      innerNonNegativeTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				yCoordinate:   thresholdNonNegativeTestFallingErrorValue,
			},
			{
				thresholdType: "WARN",
				yCoordinate:   thresholdNonNegativeTestFallingWarnValue,
			},
		},
	},
	//{
	//	name:        "Non-negative | FALLING | {limits.lowest <= error < warn <= limits.highest}",
	//	triggerType: moira.FallingTrigger,
	//	warnValue:   thresholdNonNegativeTestFallingErrorValue + thresholdTestValueIncrement,
	//	errorValue:  thresholdNonNegativeTestFallingErrorValue,
	//	limits:      innerNonNegativeTestCaseLimits,
	//	expected: []*threshold{
	//		{
	//			thresholdType: "ERROR",
	//			yCoordinate:   thresholdNonNegativeTestFallingErrorValue,
	//		},
	//	},
	//},
}

// TestGenerateThresholds tests thresholds will be generated correctly
func TestGenerateThresholds(t *testing.T) {
	thresholdTestCases := append(thresholdNegativeTestCases,
		thresholdNonNegativeTestCases...)
	for _, testCase := range thresholdTestCases {
		Convey(testCase.name, t, func(c C) {
			trigger := moira.Trigger{
				TriggerType: testCase.triggerType,
			}
			if testCase.errorValue != nil {
				errorValue := testCase.errorValue.(float64)
				trigger.ErrorValue = &errorValue
			}
			if testCase.warnValue != nil {
				warnValue := testCase.warnValue.(float64)
				trigger.WarnValue = &warnValue
			}
			limits := testCase.limits
			actual := generateThresholds(&trigger, limits)
			caseMessage := testCase.getCaseMessage()
			fmt.Println(caseMessage)
			c.So(actual, ShouldResemble, testCase.expected)
		})
	}
}
