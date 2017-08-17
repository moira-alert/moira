package checker

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

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
