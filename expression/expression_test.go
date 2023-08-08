package expression

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
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
		expression := &TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}
		result, err1 := expression.Evaluate()
		err2 := expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateOK)

		expression = &TriggerExpression{MainTargetValue: 60.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateWARN)

		expression = &TriggerExpression{MainTargetValue: 90.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateERROR)

		warnValue = 30.0
		errorValue = 10.0
		expression = &TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateOK)

		expression = &TriggerExpression{MainTargetValue: 20.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateWARN)

		expression = &TriggerExpression{MainTargetValue: 10.0, WarnValue: &warnValue, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateERROR)

		expression = &TriggerExpression{MainTargetValue: 10.0, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(err2.Error(), ShouldResemble, "error value and warning value can not be empty")
		So(result, ShouldBeEmpty)

		warnValue = 30.0
		expression = &TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira.RisingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateWARN)

		warnValue = 30.0
		expression = &TriggerExpression{MainTargetValue: 40.0, WarnValue: &warnValue, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateOK)

		errorValue = 30.0
		expression = &TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira.RisingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateERROR)

		errorValue = 30.0
		expression = &TriggerExpression{MainTargetValue: 40.0, ErrorValue: &errorValue, TriggerType: moira.FallingTrigger}
		result, err1 = expression.Evaluate()
		err2 = expression.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateOK)
	})

	Convey("Test Custom", t, func() {
		expression := "t1 > 10 && t2 > 3 ? ERROR : OK"
		trigger := &TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger}
		result, err1 := trigger.Evaluate()
		err2 := trigger.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateERROR)

		expression = "min(t1, t2) > 10 ? ERROR : OK"
		trigger = &TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger}
		result, err1 = trigger.Evaluate()
		err2 = trigger.Validate()
		So(err1, ShouldResemble, ErrInvalidExpression{fmt.Errorf("functions is forbidden")})
		So(err2, ShouldResemble, ErrInvalidExpression{fmt.Errorf("functions is forbidden")})
		So(result, ShouldBeEmpty)

		expression = "PREV_STATE"
		trigger = &TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger, PreviousState: moira.StateNODATA}
		result, err1 = trigger.Evaluate()
		err2 = trigger.Validate()
		So(err1, ShouldBeNil)
		So(err2, ShouldBeNil)
		So(result, ShouldResemble, moira.StateNODATA)

		expression = "t1 > 10 && t2 > 3 ? OK : ddd"
		trigger = &TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{"t2": 4.0}, TriggerType: moira.ExpressionTrigger}
		result, err1 = trigger.Evaluate()
		err2 = trigger.Validate()
		So(
			err1,
			ShouldResemble,
			ErrInvalidExpression{fmt.Errorf("invalid variable value: %w", fmt.Errorf("no value with name ddd"))},
		)
		So(err2, ShouldNotBeNil)
		So(result, ShouldBeEmpty)

		expression = "t1 > 10 ? OK : (t2 < 5 ? WARN : ERROR)"
		trigger = &TriggerExpression{Expression: &expression, MainTargetValue: 11.0, AdditionalTargetsValues: map[string]float64{}, TriggerType: moira.ExpressionTrigger}
		result, err1 = trigger.Evaluate()
		err2 = trigger.Validate()
		So(
			err1,
			ShouldResemble,
			ErrInvalidExpression{fmt.Errorf("invalid variable value: %w", fmt.Errorf("no value with name t2"))},
		)
		So(err2, ShouldNotBeNil)
		So(result, ShouldBeEmpty)
	})
}

func TestGetExpressionValue(t *testing.T) {
	floatVal := 10.0
	Convey("Test basic strings", t, func() {
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
			{
				name:          "ok",
				expectedValue: moira.StateOK,
			},
			{
				name:          "warn",
				expectedValue: moira.StateWARN,
			},
			{
				name:          "warning",
				expectedValue: moira.StateWARN,
			},
			{
				name:          "error",
				expectedValue: moira.StateERROR,
			},
			{
				name:          "nodata",
				expectedValue: moira.StateNODATA,
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
					values:        TriggerExpression{MainTargetValue: 11.0},
					name:          "T1",
					expectedValue: 11.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t2": 1.0}},
					name:          "t2",
					expectedValue: 1.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t2": 1.0}},
					name:          "T2",
					expectedValue: 1.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "t3",
					expectedValue: 4.0,
				},
				{
					values:        TriggerExpression{AdditionalTargetsValues: map[string]float64{"t3": 4.0, "t2": 6.0}},
					name:          "T3",
					expectedValue: 4.0,
				},
				{
					values:        TriggerExpression{PreviousState: moira.StateNODATA},
					name:          "PREV_STATE",
					expectedValue: moira.StateNODATA,
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

func TestExpressionValidate(t *testing.T) {
	warnValue := float64(60)
	errorValue := float64(90)
	Convey("Default Expressions", t, func() {
		expressions := []string{
			"t1 >= ERROR_VALUE ? ERROR : (t1 >= WARN_VALUE ? WARN : OK)",
			"t1 <= ERROR_VALUE ? ERROR : (t1 <= WARN_VALUE ? WARN : OK)",
			"t1 >= WARN_VALUE ? WARN : OK",
			"t1 >= ERROR_VALUE ? ERROR : OK",
			"t1 <= WARN_VALUE ? WARN : OK",
			"t1 <= ERROR_VALUE ? ERROR : OK",
			"WARN",
		}
		for _, expr := range expressions {
			triggerExpression := TriggerExpression{
				Expression:  &expr,
				WarnValue:   &warnValue,
				ErrorValue:  &errorValue,
				TriggerType: moira.ExpressionTrigger,
			}
			err := triggerExpression.Validate()
			So(err, ShouldBeNil)
		}
	})
	Convey("Invalids", t, func() {
		expressions := []string{
			"t1 > 10 ? ok : (t2 * 10 : ddd ? warn)",
			"t1 < WARN_VALUE ? ok ?",
			"t1 > 10 ? ok : (t2 * 10 : error ? warn)",
		}
		for _, expr := range expressions {
			triggerExpression := TriggerExpression{
				Expression:  &expr,
				WarnValue:   &warnValue,
				ErrorValue:  &errorValue,
				TriggerType: moira.ExpressionTrigger,
				AdditionalTargetsValues: map[string]float64{
					"t2": 2,
					"t3": 3,
				},
			}
			err := triggerExpression.Validate()
			t.Log(err)
			So(err, ShouldNotBeNil)
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
