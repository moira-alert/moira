package expression

import (
	"testing"

	"github.com/moira-alert/moira/expression"
)

func BenchmarkDefault1Expr(b *testing.B) {
	warnValue := 60.0
	errorValue := 90.0
	expr := &expression.TriggerExpression{
		MainTargetValue: 10.0,
		WarnValue:       &warnValue,
		ErrorValue:      &errorValue,
	}
	for i := 0; i < b.N; i++ {
		(expr).Evaluate() //nolint
	}
}

func BenchmarkDefault2Expr(b *testing.B) {
	warnValue := 90.0
	errorValue := 60.0
	expr := &expression.TriggerExpression{
		MainTargetValue: 10.0,
		WarnValue:       &warnValue,
		ErrorValue:      &errorValue,
	}
	for i := 0; i < b.N; i++ {
		(expr).Evaluate() //nolint
	}
}

func BenchmarkCustomExpr(b *testing.B) {
	expressionStr := "t1 > 10 && t2 > 3 ? ERROR : OK"
	expr := &expression.TriggerExpression{
		Expression:              &expressionStr,
		MainTargetValue:         11.0,
		AdditionalTargetsValues: map[string]float64{"t2": 4.0},
	}
	for i := 0; i < b.N; i++ {
		(expr).Evaluate() //nolint
	}
}
