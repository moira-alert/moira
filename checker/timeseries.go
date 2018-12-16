package checker

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/remote"
	"github.com/moira-alert/moira/target"
)

// TriggerTimeSeries represent collection of Main target timeseries
// and collection of additions targets timeseries
type TriggerTimeSeries struct {
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

// GetTriggerEvaluationResult to test checker
func GetTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*TriggerTimeSeries, *moira.Trigger, error) {
	allowRealtimeAlerting := true
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return nil, nil, err
	}
	triggerMetrics := &TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	if trigger.IsRemote && !remoteConfig.IsEnabled() {
		return nil, &trigger, remote.ErrRemoteStorageDisabled
	}
	for i, tar := range trigger.Targets {
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAlerting)
			if err != nil {
				return nil, &trigger, err
			}
			timeSeries = result.TimeSeries
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, &trigger, nil
}

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*TriggerTimeSeries, []string, error) {
	wrongTriggerTargets := make([]int, 0)

	triggerTimeSeries := &TriggerTimeSeries{
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
					return nil, nil, fmt.Errorf("target t%v has no timeseries", targetIndex+1)
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

func (triggerChecker *TriggerChecker) getRemoteTimeSeries(from, until int64) (*TriggerTimeSeries, error) {
	wrongTriggerTargets := make([]int, 0)

	triggerTimeSeries := &TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}

	isSimpleTrigger := triggerChecker.trigger.IsSimple()
	for targetIndex, tar := range triggerChecker.trigger.Targets {
		timeSeries, err := remote.Fetch(triggerChecker.RemoteConfig, tar, from, until, isSimpleTrigger)
		if err != nil {
			return nil, err
		}

		if targetIndex == 0 {
			triggerTimeSeries.Main = timeSeries
		} else {
			timeSeriesCount := len(timeSeries)
			switch {
			case timeSeriesCount == 0:
				return nil, fmt.Errorf("target t%v has no timeseries", targetIndex+1)
			case timeSeriesCount > 1:
				wrongTriggerTargets = append(wrongTriggerTargets, targetIndex+1)
			default: // == 1
				triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, timeSeries[0])
			}
		}
	}

	if len(wrongTriggerTargets) > 0 {
		return nil, ErrWrongTriggerTargets(wrongTriggerTargets)
	}

	return triggerTimeSeries, nil
}

func (*TriggerTimeSeries) getMainTargetName() string {
	return "t1"
}

func (*TriggerTimeSeries) getAdditionalTargetName(targetIndex int) string {
	return fmt.Sprintf("t%v", targetIndex+2)
}

func (triggerTimeSeries *TriggerTimeSeries) getExpressionValues(firstTargetTimeSeries *target.TimeSeries, valueTimestamp int64) (*expression.TriggerExpression, bool) {
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
func (triggerTimeSeries *TriggerTimeSeries) hasOnlyWildcards() bool {
	for _, timeSeries := range triggerTimeSeries.Main {
		if !timeSeries.Wildcard {
			return false
		}
	}
	return true
}
