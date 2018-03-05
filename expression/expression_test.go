package expression

import (
	"fmt"
	"testing"

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
		result, err := (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "OK")

		result, err = (&TriggerExpression{MainTargetValue: 60.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "WARN")

		result, err = (&TriggerExpression{MainTargetValue: 90.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "ERROR")

		warnValue = 30.0
		errorValue = 10.0
		result, err = (&TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "OK")

		result, err = (&TriggerExpression{MainTargetValue: 20.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "WARN")

		result, err = (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "ERROR")

		result, err = (&TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue}).Evaluate()
		So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("error value and Warning value can not be empty")})
		So(err.Error(), ShouldResemble, "error value and Warning value can not be empty")
		So(result, ShouldBeEmpty)
	})

	Convey("Test Custom", t, func() {
		expression := "t1 > 10 && t2 > 3 ? ERROR : OK"
		result, err := (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}}).Evaluate()
		So(err, ShouldBeNil)
		So(result, ShouldResemble, "ERROR")

		expression = "min(t1, t2) > 10 ? ERROR : OK"
		result, err = (&TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}}).Evaluate()
		So(err, ShouldResemble, ErrInvalidExpression{fmt.Errorf("functions is forbidden")})
		So(result, ShouldBeEmpty)
	})
}

func TestGetExpressionValue(t *testing.T) {
	floatVal := 10.0
	Convey("Test basic strings", t, func() {
		getExpressionValuesTests := []getExpressionValuesTest{
			{
				name:          "OK",
				expectedValue: "OK",
			},
			{
				name:          "WARN",
				expectedValue: "WARN",
			},
			{
				name:          "WARNING",
				expectedValue: "WARN",
			},
			{
				name:          "ERROR",
				expectedValue: "ERROR",
			},
			{
				name:          "NODATA",
				expectedValue: "NODATA",
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
					values:        TriggerExpression{PreviousState: "NODATA"},
					name:          "PREV_STATE",
					expectedValue: "NODATA",
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
