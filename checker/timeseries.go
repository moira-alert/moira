package checker

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"math"
)

type TimeSeries expr.MetricData

type triggerTimeSeries struct {
	Main       []*TimeSeries
	Additional []*TimeSeries
}

func (*triggerTimeSeries) getMainTargetName() string {
	return "t1"
}

func (*triggerTimeSeries) getAdditionalTargetName(targetNumber int) string {
	return fmt.Sprintf("t%v", targetNumber+2)
}

func (triggerTimeSeries *triggerTimeSeries) getExpressionValues(firstTargetTimeSeries *TimeSeries, checkPoint int64) (ExpressionValues, bool) {
	expressionValues := make(map[string]float64)
	firstTargetValue := firstTargetTimeSeries.getCheckPointValue(checkPoint)
	if math.IsNaN(firstTargetValue) {
		return expressionValues, false
	}
	expressionValues[triggerTimeSeries.getMainTargetName()] = firstTargetValue

	for targetNumber := 0; targetNumber < len(triggerTimeSeries.Additional); targetNumber++ {
		additionalTimeSeries := triggerTimeSeries.Additional[targetNumber]
		if additionalTimeSeries == nil {
			return expressionValues, false
		}
		tnValue := additionalTimeSeries.getCheckPointValue(checkPoint)
		if math.IsNaN(tnValue) {
			return expressionValues, false
		}
		expressionValues[triggerTimeSeries.getAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}

func (timeSeries *TimeSeries) getCheckPointValue(checkPoint int64) float64 {
	valueIndex := int((checkPoint - int64(timeSeries.StartTime)) / int64(timeSeries.StepTime))
	if len(timeSeries.Values) <= valueIndex || (len(timeSeries.IsAbsent) > valueIndex && timeSeries.IsAbsent[valueIndex]) {
		return math.NaN()
	} else {
		return timeSeries.Values[valueIndex]
	}
}
