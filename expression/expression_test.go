package expression

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

type getExpressionValuesTest struct {
	values        TriggerExpression
	name          string
	expectedError error
	expectedValue interface{}
}

func TestExpression(t *testing.T) {
	Convey("Test Default", t, func(c C) {
		warnValue := 60.0
		errorValue := 90.0
		result, err := (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateOK)

		result, err = (&TriggerExpression{MainTargetValue: 60.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateWARN)

		result, err = (&TriggerExpression{MainTargetValue: 90.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateERROR)

		warnValue = 30.0
		errorValue = 10.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateOK)

		result, err = (&TriggerExpression{MainTargetValue: 20.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateWARN)

		result, err = (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateERROR)

		result, err = (&TriggerExpression{MainTargetValue: 10.0, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("error value and warning value can not be empty")})
		c.So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		c.So(result, ShouldBeEmpty)

		warnValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira.RisingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateWARN)

		warnValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateOK)

		errorValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateERROR)

		errorValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateOK)
	})

	Convey("Test Custom", t, func(c C) {
		expression := "t1 > 10 && t2 > 3 ? ERROR : OK"
		result, err := (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateERROR)

		expression = "min(t1, t2) > 10 ? ERROR : OK"
		result, err = (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger}).Evaluate()
		c.So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("functions is forbidden")})
		c.So(result, ShouldBeEmpty)

		expression = "PREV_STATE"
		result, err = (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger, PreviousState: moira.StateNODATA}).Evaluate()
		c.So(err, ShouldBeNil)
		c.So(result, ShouldResemble, moira.StateNODATA)
	})
}

func TestGetExpressionValue(t *testing.T) {
	floatVal := 10.0
	Convey("Test basic strings", t, func(c C) {
		getExpressionValuesTests := []getExpressionValuesTest{
			{
				name:          "OK",
				expectedValue: moira.StateOK,
			},
			{
				name:          "WARN",
				expectedValue: moira.StateWARN,
			},
			{
				name:          "WARNING",
				expectedValue: moira.StateWARN,
			},
			{
				name:          "ERROR",
				expectedValue: moira.StateERROR,
			},
			{
				name:          "NODATA",
				expectedValue: moira.StateNODATA,
			},
		}
		runGetExpressionValuesTest(c, getExpressionValuesTests)
	})

	Convey("Test no errors", t, func(c C) {
		{
			getExpressionValuesTests := []getExpressionValuesTest{
				{
					values:        TriggerExpression{WarnValue: &floatVal},
					name:          "WARN_VALUE",
					expectedValue: floatVal,
				},
				{
					values:        TriggerExpression{ErrorValue: &floatVal},
					name:          "ERROR_VALUE",
					expectedValue: floatVal,
				},
				{
					values:        TriggerExpression{MainTargetValue: 11.0},
					name:          "t1",
					expectedValue: 11.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t2": 1.0}},
					name:          "t2",
					expectedValue: 1.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "t3",
					expectedValue: 4.0,
				},
				{
					values:        TriggerExpression{PreviousState: moira.StateNODATA},
					name:          "PREV_STATE",
					expectedValue: moira.StateNODATA,
				},
			}
			runGetExpressionValuesTest(c, getExpressionValuesTests)
		}
	})

	Convey("Test errors", t, func(c C) {
		{
			getExpressionValuesTests := []getExpressionValuesTest{
				{
					name:          "WARN_VALUE",
					expectedValue: nil,
					expectedError: fmt.Errorf("no value with name WARN_VALUE"),
				},
				{
					name:          "ERROR_VALUE",
					expectedValue: nil,
					expectedError: fmt.Errorf("no value with name ERROR_VALUE"),
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "t4",
					expectedValue: nil,
					expectedError: fmt.Errorf("no value with name t4"),
				},
			}
			runGetExpressionValuesTest(c, getExpressionValuesTests)
		}
	})
}

func runGetExpressionValuesTest(c C, getExpressionValuesTests []getExpressionValuesTest) {
	for _, getExpressionValuesTest := range getExpressionValuesTests {
		result, err := getExpressionValuesTest.values.Get(getExpressionValuesTest.name)
		c.So(err, ShouldResemble, getExpressionValuesTest.expectedError)
		c.So(result, ShouldResemble, getExpressionValuesTest.expectedValue)
	}
}
