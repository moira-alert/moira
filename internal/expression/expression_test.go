package expression

import (
	"fmt"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	. "github.com/smartystreets/goconvey/convey"
)

type getExpressionValuesTest struct {
	values        TriggerExpression
	name          string
	expectedError error
	expectedValue interface{}
}

func TestExpression(t *testing.T) {
	Convey("Test Default", t, func() {
		warnValue := 60.0
		errorValue := 90.0
		result, err := (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.RisingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateOK)

		result, err = (&TriggerExpression{MainTargetValue: 60.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.RisingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateWARN)

		result, err = (&TriggerExpression{MainTargetValue: 90.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.RisingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateERROR)

		warnValue = 30.0
		errorValue = 10.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateOK)

		result, err = (&TriggerExpression{MainTargetValue: 20.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateWARN)

		result, err = (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateERROR)

		result, err = (&TriggerExpression{MainTargetValue: 10.0, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("error value and warning value can not be empty")})
		So(err.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(result, ShouldBeEmpty)

		warnValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira2.RisingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateWARN)

		warnValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateOK)

		errorValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira2.RisingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateERROR)

		errorValue = 30.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira2.FallingTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateOK)
	})

	Convey("Test Custom", t, func() {
		expression := "t1 > 10 && t2 > 3 ? ERROR : OK"
		result, err := (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira2.ExpressionTrigger}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateERROR)

		expression = "min(t1, t2) > 10 ? ERROR : OK"
		result, err = (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira2.ExpressionTrigger}).Evaluate()
		So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("functions is forbidden")})
		So(result, ShouldBeEmpty)

		expression = "PREV_STATE"
		result, err = (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira2.ExpressionTrigger, PreviousState: moira2.StateNODATA}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, moira2.StateNODATA)
	})
}

func TestGetExpressionValue(t *testing.T) {
	floatVal := 10.0
	Convey("Test basic strings", t, func() {
		getExpressionValuesTests := []getExpressionValuesTest{
			{
				name:          "OK",
				expectedValue: moira2.StateOK,
			},
			{
				name:          "WARN",
				expectedValue: moira2.StateWARN,
			},
			{
				name:          "WARNING",
				expectedValue: moira2.StateWARN,
			},
			{
				name:          "ERROR",
				expectedValue: moira2.StateERROR,
			},
			{
				name:          "NODATA",
				expectedValue: moira2.StateNODATA,
			},
		}
		runGetExpressionValuesTest(getExpressionValuesTests)
	})

	Convey("Test no errors", t, func() {
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
					values:        TriggerExpression{PreviousState: moira2.StateNODATA},
					name:          "PREV_STATE",
					expectedValue: moira2.StateNODATA,
				},
			}
			runGetExpressionValuesTest(getExpressionValuesTests)
		}
	})

	Convey("Test errors", t, func() {
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
			runGetExpressionValuesTest(getExpressionValuesTests)
		}
	})
}

func runGetExpressionValuesTest(getExpressionValuesTests []getExpressionValuesTest) {
	for _, getExpressionValuesTest := range getExpressionValuesTests {
		result, err := getExpressionValuesTest.values.Get(getExpressionValuesTest.name)
		So(err, ShouldResemble, getExpressionValuesTest.expectedError)
		So(result, ShouldResemble, getExpressionValuesTest.expectedValue)
	}
}
