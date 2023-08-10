package expression

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
)

func BenchmarkDefault1Expr(b *testing.B) {
	warnValue := 60.0
	errorValue := 90.0
	expr := &expression.TriggerExpression{
		MainTargetValue: 10.0,
		WarnValue:       &warnValue,
		ErrorValue:      &errorValue,
		TriggerType:     moira.RisingTrigger,
	}
	for i := 0; i < b.N; i++ {
		_, err := expr.Evaluate()
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkDefault2Expr(b *testing.B) {
	warnValue := 90.0
	errorValue := 60.0
	expr := &expression.TriggerExpression{
		MainTargetValue: 10.0,
		WarnValue:       &warnValue,
		ErrorValue:      &errorValue,
		TriggerType:     moira.RisingTrigger,
	}
	for i := 0; i < b.N; i++ {
		_, err := expr.Evaluate()
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkCustomExpr(b *testing.B) {
	expressionStr := "t1 > 10 && t2 > 3 ? ERROR : OK"
	expr := &expression.TriggerExpression{
		Expression:              &expressionStr,
		TriggerType:             moira.ExpressionTrigger,
		MainTargetValue:         11.0,
		AdditionalTargetsValues: map[string]float64{"t2": 4.0}}
	for i := 0; i < b.N; i++ {
		_, err := expr.Evaluate()
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkValidateComplex(b *testing.B) {
	expressionStr := "(t1 * 2 > t2 && t2 / 2 != 0) || (t3 * t4 == t5 && t6 < t7) ? (t8 > t9 ? OK : WARN) : ERROR"
	expr := &expression.TriggerExpression{
		Expression:      &expressionStr,
		TriggerType:     moira.ExpressionTrigger,
		MainTargetValue: 4,
		AdditionalTargetsValues: map[string]float64{
			"t2": 5,
			"t3": 3,
			"t4": 6,
			"t5": 18,
			"t6": 10,
			"t7": 15,
			"t8": 20,
			"t9": 10,
		},
	}
	for i := 0; i < b.N; i++ {
		err := expr.Validate()
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkEvaluateComplex(b *testing.B) {
	expressionStr := "(t1 * 2 > t2 && t2 / 2 != 0) || (t3 * t4 == t5 && t6 < t7) ? (t8 > t9 ? OK : WARN) : ERROR"
	expr := &expression.TriggerExpression{
		Expression:      &expressionStr,
		TriggerType:     moira.ExpressionTrigger,
		MainTargetValue: 4,
		AdditionalTargetsValues: map[string]float64{
			"t2": 5,
			"t3": 3,
			"t4": 6,
			"t5": 18,
			"t6": 10,
			"t7": 15,
			"t8": 20,
			"t9": 10,
		},
	}
	for i := 0; i < b.N; i++ {
		_, err := expr.Evaluate()
		if err != nil {
			b.Log(err)
		}
	}
}
