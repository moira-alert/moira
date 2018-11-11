package plotting

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const (
	thresholdTestValueIncrement    = float64(10)
	thresholdTestRisingWarnValue   = float64(100)
	thresholdTestRisingErrorValue  = float64(200)
	thresholdTestFallingWarnValue  = thresholdTestRisingErrorValue
	thresholdTestFallingErrorValue = thresholdTestRisingWarnValue
)

var (
	innerTestCaseLimits = plotLimits{
		from:    int64ToTime(0),
		to:      int64ToTime(100),
		lowest:  0,
		highest: 200,
	}
	outerTestCaseLimits = plotLimits{
		from:    int64ToTime(0),
		to:      int64ToTime(100),
		lowest:  1000,
		highest: 2000,
	}
)

type thresholdTestCase struct {
	name        string
	triggerType string
	warnValue   interface{}
	errorValue  interface{}
	limits      plotLimits
	expected    []*threshold
}

var thresholdTestCases = []thresholdTestCase{
	{
		name:        "RISING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdTestRisingWarnValue,
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdTestRisingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, warn, error",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdTestRisingWarnValue,
		errorValue:  thresholdTestRisingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdTestRisingWarnValue,
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - thresholdTestRisingWarnValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  thresholdTestRisingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - thresholdTestRisingErrorValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn << error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdTestRisingWarnValue,
		errorValue:  thresholdTestRisingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - thresholdTestRisingErrorValue,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - thresholdTestRisingWarnValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn < error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   thresholdTestRisingWarnValue,
		errorValue:  thresholdTestRisingWarnValue + thresholdTestValueIncrement,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - (thresholdTestRisingWarnValue + thresholdTestValueIncrement),
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdTestFallingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdTestFallingWarnValue,
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdTestFallingWarnValue,
		errorValue:  thresholdTestFallingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  thresholdTestFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   thresholdTestFallingErrorValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdTestFallingWarnValue,
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   thresholdTestFallingWarnValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error << warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdTestFallingWarnValue,
		errorValue:  thresholdTestFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   thresholdTestFallingErrorValue,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   thresholdTestFallingWarnValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error < warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   thresholdTestFallingErrorValue + thresholdTestValueIncrement,
		errorValue:  thresholdTestFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   thresholdTestFallingErrorValue,
			},
		},
	},
}

// TestGenerateThresholds tests thresholds will be generated correctly
func TestGenerateThresholds(t *testing.T) {
	for _, testCase := range thresholdTestCases {
		Convey(testCase.name, t, func() {
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
			So(actual, ShouldResemble, testCase.expected)
		})
	}
}
