package checker

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/target"
)

type triggerTimeSeries struct {
	Main       []*target.TimeSeries
	Additional []*target.TimeSeries
}

// ErrWrongTriggerTargets represents targets with inconsistent number of timeseries
type ErrWrongTriggerTargets []int

// ErrWrongTriggerTarget implementation for list of invalid targets found
func (err ErrWrongTriggerTargets) Error() string {
	var countType []byte
	if len(err) > 1 {
		countType = []byte("Targets ")
	} else {
		countType = []byte("Target ")
	}
	wrongTargets := bytes.NewBuffer(countType)
	for tarInd, tar := range err {
		wrongTargets.WriteString("t")
		wrongTargets.WriteString(strconv.Itoa(tar))
		if tarInd != len(err)-1 {
			wrongTargets.WriteString(", ")
		}
	}
	wrongTargets.WriteString(" has more than one timeseries")
	return wrongTargets.String()
}

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*triggerTimeSeries, []string, error) {
	wrongTriggerTargets := make([]int, 0)

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
			timeSeriesCount := len(result.TimeSeries)
			switch {
			case timeSeriesCount == 0:
				if len(result.Metrics) == 0 {
					triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, nil)
				} else {
					return nil, nil, fmt.Errorf("Target t%v has no timeseries", targetIndex+1)
				}
			case timeSeriesCount > 1:
				wrongTriggerTargets = append(wrongTriggerTargets, targetIndex+1)
			default:
				triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, result.TimeSeries[0])
			}
		}
		metricsArr = append(metricsArr, result.Metrics...)
	}

	if len(wrongTriggerTargets) > 0 {
		return nil, nil, ErrWrongTriggerTargets(wrongTriggerTargets)
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
