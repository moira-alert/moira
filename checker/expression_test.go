package checker

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type getExpressionValuesTest struct {
	values        ExpressionValues
	name          string
	expectedError error
	expectedValue interface{}
}

func TestExpression(t *testing.T) {
	Convey("Test Default", t, func() {
		var warnValue float64 = 60.0
		var errorValue float64 = 90.0
		result, err := EvaluateExpression(nil, ExpressionValues{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, OK)

		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 60.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, WARN)

		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 90.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, ERROR)

		warnValue = 30.0
		errorValue = 10.0
		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 40.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, OK)

		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 20.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, WARN)

		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, ERROR)

		result, err = EvaluateExpression(nil, ExpressionValues{MainTargetValue: 10.0, WarnValue: &warnValue})
		So(err, ShouldResemble, fmt.Errorf("Error value and Warning value can not be empty"))
		So(result, ShouldBeEmpty)
	})

	Convey("Test Custom", t, func() {
		expression := "t1 > 10 && t2 > 3 ? ERROR : OK"
		result, err := EvaluateExpression(&expression, ExpressionValues{MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}})
		So(err, ShouldBeNil)
		So(result, ShouldResemble, ERROR)

		expression = "min(t1, t2) > 10 ? ERROR : OK"
		result, err = EvaluateExpression(&expression, ExpressionValues{MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}})
		So(err, ShouldResemble, fmt.Errorf("Functions is forbidden"))
		So(result, ShouldBeEmpty)
	})
}

func TestGetExpressionValue(t *testing.T) {
	var floatVal float64 = 10

	Convey("Test basic strings", t, func() {
		getExpressionValuesTests := []getExpressionValuesTest{
			{
				name:          "OK",
				expectedValue: OK,
			},
			{
				name:          "WARN",
				expectedValue: WARN,
			},
			{
				name:          "WARNING",
				expectedValue: WARN,
			},
			{
				name:          "ERROR",
				expectedValue: ERROR,
			},
			{
				name:          "NODATA",
				expectedValue: NODATA,
			},
		}
		runGetExpressionValuesTest(getExpressionValuesTests)
	})

	Convey("Test no errors", t, func() {
		{
			getExpressionValuesTests := []getExpressionValuesTest{
				{
					values:        ExpressionValues{WarnValue: &floatVal},
					name:          "WARN_VALUE",
					expectedValue: floatVal,
				},
				{
					values:        ExpressionValues{ErrorValue: &floatVal},
					name:          "ERROR_VALUE",
					expectedValue: floatVal,
				},
				{
					values:        ExpressionValues{MainTargetValue: 11.0},
					name:          "t1",
					expectedValue: 11.0,
				},
				{
					values:        ExpressionValues{AdditionalTargetsValues: map[string]float64{"t2": 1.0}},
					name:          "t2",
					expectedValue: 1.0,
				},
				{
					values:        ExpressionValues{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "t3",
					expectedValue: 4.0,
				},
				{
					values:        ExpressionValues{PreviousState: NODATA},
					name:          "PREV_STATE",
					expectedValue: NODATA,
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
					expectedError: fmt.Errorf("No value with name WARN_VALUE"),
				},
				{
					name:          "ERROR_VALUE",
					expectedValue: nil,
					expectedError: fmt.Errorf("No value with name ERROR_VALUE"),
				},
				{
					values:        ExpressionValues{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "t4",
					expectedValue: nil,
					expectedError: fmt.Errorf("No value with name t4"),
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
