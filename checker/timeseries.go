package checker

import (
	"fmt"
	"math"

	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/target"
)

type triggerTimeSeries struct {
	Main       []*target.TimeSeries
	Additional []*target.TimeSeries
}

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*triggerTimeSeries, []string, error) {
	triggerTimeSeries := &triggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	metricsArr := make([]string, 0)

	isSimpleTrigger := triggerChecker.trigger.IsSimple()
	for targetIndex, tar := range triggerChecker.trigger.Targets {
		result, err := target.EvaluateTarget(triggerChecker.Database, tar, from, until, isSimpleTrigger)
		if err != nil {
			return nil, nil, err
		}

		if targetIndex == 0 {
			triggerTimeSeries.Main = result.TimeSeries
		} else {
			if len(result.TimeSeries) == 0 && len(result.Metrics) != 0 {
				return nil, nil, fmt.Errorf("Target #%v has no timeseries", targetIndex+1)
			} else if len(result.TimeSeries) > 1 {
				return nil, nil, fmt.Errorf("Target #%v has more than one timeseries", targetIndex+1)
			} else if len(result.TimeSeries) == 0 {
				triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, nil)
			} else {
				triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, result.TimeSeries[0])
			}
		}
		metricsArr = append(metricsArr, result.Metrics...)
	}
	return triggerTimeSeries, metricsArr, nil
}

func (*triggerTimeSeries) getMainTargetName() string {
	return "t1"
}

func (*triggerTimeSeries) getAdditionalTargetName(targetIndex int) string {
	return fmt.Sprintf("t%v", targetIndex+2)
}

func (triggerTimeSeries *triggerTimeSeries) getExpressionValues(firstTargetTimeSeries *target.TimeSeries, valueTimestamp int64) (*expression.TriggerExpression, bool) {
	expressionValues := &expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64, len(triggerTimeSeries.Additional)),
	}
	firstTargetValue := firstTargetTimeSeries.GetTimestampValue(valueTimestamp)
	if IsInvalidValue(firstTargetValue) {
		return expressionValues, false
	}
	expressionValues.MainTargetValue = firstTargetValue

	for targetNumber := 0; targetNumber < len(triggerTimeSeries.Additional); targetNumber++ {
		additionalTimeSeries := triggerTimeSeries.Additional[targetNumber]
		if additionalTimeSeries == nil {
			return expressionValues, false
		}
		tnValue := additionalTimeSeries.GetTimestampValue(valueTimestamp)
		if IsInvalidValue(tnValue) {
			return expressionValues, false
		}
		expressionValues.AdditionalTargetsValues[triggerTimeSeries.getAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}

// IsInvalidValue checks trigger for Inf and NaN. If it is then trigger is not valid
func IsInvalidValue(val float64) bool {
	if math.IsNaN(val) {
		return true
	}
	if math.IsInf(val, 0) {
		return true
	}
	return false
}

// hasOnlyWildcards checks given targetTimeSeries for only wildcards
func (triggerTimeSeries *triggerTimeSeries) hasOnlyWildcards() bool {
	for _, timeSeries := range triggerTimeSeries.Main {
		if !timeSeries.Wildcard {
			return false
		}
	}
	return true
}
