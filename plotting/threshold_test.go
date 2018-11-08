package plotting

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
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
		warnValue:   float64(100),
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  float64(200),
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest ..limits.highest}, warn, error",
		triggerType: moira.RisingTrigger,
		warnValue:   float64(100),
		errorValue:  float64(200),
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "RISING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   float64(100),
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - 100,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   nil,
		errorValue:  float64(200),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - 200,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn << error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   float64(100),
		errorValue:  float64(200),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - 200,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - 100,
			},
		},
	},
	{
		name:        "RISING | {limits.lowest <= warn < error <= limits.highest}",
		triggerType: moira.RisingTrigger,
		warnValue:   float64(100),
		errorValue:  float64(110),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   innerTestCaseLimits.highest - 110,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  float64(100),
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   float64(200),
		errorValue:  nil,
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest ..limits.highest}, error, warn",
		triggerType: moira.FallingTrigger,
		warnValue:   float64(200),
		errorValue:  float64(100),
		limits:      outerTestCaseLimits,
		expected:    []*threshold{},
	},
	{
		name:        "FALLING | {limits.lowest <= error <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   nil,
		errorValue:  float64(100),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   100,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   float64(200),
		errorValue:  nil,
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   200,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error << warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   float64(200),
		errorValue:  float64(100),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   100,
			},
			{
				thresholdType: "WARN",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   200,
			},
		},
	},
	{
		name:        "FALLING | {limits.lowest <= error < warn <= limits.highest}",
		triggerType: moira.FallingTrigger,
		warnValue:   float64(110),
		errorValue:  float64(100),
		limits:      innerTestCaseLimits,
		expected: []*threshold{
			{
				thresholdType: "ERROR",
				xCoordinate:   float64(innerTestCaseLimits.to.UnixNano()),
				yCoordinate:   100,
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
