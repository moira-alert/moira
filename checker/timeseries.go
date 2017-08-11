package checker

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
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

func (targetTimeSeries triggerTimeSeries) getExpressionValues(firstTargetTimeSeries *TimeSeries, checkPoint int64) (ExpressionValues, bool) {
	expressionValues := make(map[string]float64)
	firstTargetValue := firstTargetTimeSeries.getTimeSeriesCheckPointValue(checkPoint)
	if math.IsNaN(firstTargetValue) {
		return expressionValues, false
	}
	expressionValues[targetTimeSeries.getMainTargetName()] = firstTargetValue

	for targetNumber := 0; targetNumber <= len(targetTimeSeries.Additional); targetNumber++ {
		additionalTimeSeries := targetTimeSeries.Additional[targetNumber]
		if additionalTimeSeries == nil {
			return expressionValues, false
		}
		tnValue := additionalTimeSeries.getTimeSeriesCheckPointValue(checkPoint)
		if math.IsNaN(tnValue) {
			return expressionValues, false
		}
		expressionValues[targetTimeSeries.getAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}

func (targetTimeSeries *triggerTimeSeries) updateCheckData(firstTargetTimeSeries *TimeSeries, checkData *moira.CheckData, expressionState string, expressionValues ExpressionValues, valueTimestamp int64) {
	metricState := checkData.Metrics[firstTargetTimeSeries.Name]
	metricState.State = expressionState
	metricState.Timestamp = valueTimestamp

	if len(expressionValues) == 0 {
		if metricState.Value != nil {
			metricState.Value = nil
		}
	} else {
		val := expressionValues[targetTimeSeries.getMainTargetName()]
		metricState.Value = &val
	}

	checkData.Metrics[firstTargetTimeSeries.Name] = metricState
}

func (timeSeries *TimeSeries) getTimeSeriesCheckPointValue(checkPoint int64) float64 {
	valueIndex := (checkPoint - int64(timeSeries.StartTime)) / int64(timeSeries.StepTime)
	var value float64
	if len(timeSeries.Values) > int(valueIndex) {
		value = timeSeries.Values[valueIndex]
	} else {
		value = math.NaN()
	}
	return value
}
