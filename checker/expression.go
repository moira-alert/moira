package checker

type ExpressionValues map[string]float64

func (values ExpressionValues) GetTargetValue(targetName string) *float64 {
	if len(values) == 0 {
		return nil
	}
	val := values[targetName]
	return &val
}

func GetExpression(triggerExpression *string, expressionValues map[string]float64) string {
	return OK
}
