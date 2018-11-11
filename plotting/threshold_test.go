package plotting

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const (
	testValueIncrement    = float64(10)
	testRisingWarnValue   = float64(100)
	testRisingErrorValue  = float64(200)
	testFallingWarnValue  = testRisingErrorValue
	testFallingErrorValue = testRisingWarnValue
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
		warnValue:   testRisingWarnValue,
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  testRisingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, warn, error",
		triggerType: moira.RisingTrigger,
		warnValue:   testRisingWarnValue,
		errorValue:  testRisingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   testRisingWarnValue,
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - testRisingWarnValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  testRisingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - testRisingErrorValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn << error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   testRisingWarnValue,
		errorValue:  testRisingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - testRisingErrorValue,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - testRisingWarnValue,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn < error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   testRisingWarnValue,
		errorValue:  testRisingWarnValue + testValueIncrement,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - (testRisingWarnValue + testValueIncrement),
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  testFallingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   testFallingWarnValue,
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   testFallingWarnValue,
		errorValue:  testFallingErrorValue,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  testFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   testFallingErrorValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   testFallingWarnValue,
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   testFallingWarnValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error << warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   testFallingWarnValue,
		errorValue:  testFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   testFallingErrorValue,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   testFallingWarnValue,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error < warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   testFallingErrorValue + testValueIncrement,
		errorValue:  testFallingErrorValue,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   testFallingErrorValue,
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
