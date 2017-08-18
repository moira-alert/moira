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

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*triggerTimeSeries, []string, error) {
	triggerTimeSeries := &triggerTimeSeries{
		Main:       make([]*TimeSeries, 0),
		Additional: make([]*TimeSeries, 0),
	}
	metricsArr := make([]string, 0)

	for targetIndex, target := range triggerChecker.trigger.Targets {
		result, err := EvaluateTarget(triggerChecker.Database, target, from, until, triggerChecker.isSimple)
		if err != nil {
			return nil, nil, err
		}

		if targetIndex == 0 {
			triggerTimeSeries.Main = result.TimeSeries
		} else {
			if len(result.TimeSeries) == 0 {
				return nil, nil, fmt.Errorf("Target #%v has no timeseries", targetIndex+1)
			} else if len(result.TimeSeries) > 1 {
				return nil, nil, fmt.Errorf("Target #%v has more than one timeseries", targetIndex+1)
			}
			triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, result.TimeSeries[0])
		}
		metricsArr = append(metricsArr, result.Metrics...)
	}
	return triggerTimeSeries, metricsArr, nil
}

func (*triggerTimeSeries) getMainTargetName() string {
	return "t1"
}

func (*triggerTimeSeries) getAdditionalTargetName(targetNumber int) string {
	return fmt.Sprintf("t%v", targetNumber+2)
}

func (triggerTimeSeries *triggerTimeSeries) getExpressionValues(firstTargetTimeSeries *TimeSeries, valueTimestamp int64) (ExpressionValues, bool) {
	expressionValues := ExpressionValues{
		AdditionalTargetsValues: make(map[string]float64),
	}
	firstTargetValue := firstTargetTimeSeries.getTimestampValue(valueTimestamp)
	if math.IsNaN(firstTargetValue) {
		return expressionValues, false
	}
	expressionValues.MainTargetValue = firstTargetValue

	for targetNumber := 0; targetNumber < len(triggerTimeSeries.Additional); targetNumber++ {
		additionalTimeSeries := triggerTimeSeries.Additional[targetNumber]
		if additionalTimeSeries == nil {
			return expressionValues, false
		}
		tnValue := additionalTimeSeries.getTimestampValue(valueTimestamp)
		if math.IsNaN(tnValue) {
			return expressionValues, false
		}
		expressionValues.AdditionalTargetsValues[triggerTimeSeries.getAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}

func (timeSeries *TimeSeries) getTimestampValue(valueTimestamp int64) float64 {
	if valueTimestamp < int64(timeSeries.StartTime) {
		return math.NaN()
	}
	valueIndex := int((valueTimestamp - int64(timeSeries.StartTime)) / int64(timeSeries.StepTime))
	if len(timeSeries.IsAbsent) > valueIndex && timeSeries.IsAbsent[valueIndex] {
		return math.NaN()
	}
	if len(timeSeries.Values) <= valueIndex {
		return math.NaN()
	}
	return timeSeries.Values[valueIndex]
}
