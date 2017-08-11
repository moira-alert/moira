package checker

import (
	"fmt"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira-alert"
	"math"
)

type TimeSeries expr.MetricData

type triggerTimeSeries map[int][]*TimeSeries

func (targetTimeSeries triggerTimeSeries) getExpressionValues(firstTargetTimeSeries *TimeSeries, checkPoint int64) (map[string]float64, bool) {
	expressionValues := make(map[string]float64)
	firstTargetValue := firstTargetTimeSeries.getTimeSeriesCheckPointValue(checkPoint)
	if math.IsNaN(firstTargetValue) {
		return expressionValues, false
	}

	for targetNumber := 2; targetNumber <= len(targetTimeSeries); targetNumber++ {
		if len(targetTimeSeries[targetNumber]) == 0 {
			return expressionValues, false
		}
		tN := targetTimeSeries[targetNumber][0]
		tnValue := tN.getTimeSeriesCheckPointValue(checkPoint)
		if math.IsNaN(tnValue) {
			break
		}
		targetName := fmt.Sprintf("t%v", targetNumber)
		expressionValues[targetName] = tnValue
	}
	return expressionValues, true
}

func (targetTimeSeries *triggerTimeSeries) updateCheckData(firstTargetTimeSeries *TimeSeries, checkData *moira.CheckData, expressionState string, expressionValues map[string]float64, valueTimestamp int64) {
	metricState := checkData.Metrics[firstTargetTimeSeries.Name]
	metricState.State = expressionState
	metricState.Timestamp = valueTimestamp

	if len(expressionValues) == 0 {
		if metricState.Value != nil {
			metricState.Value = nil
		}
	} else {
		val := expressionValues["t1"]
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
