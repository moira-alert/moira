package checker

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"
)

var (
	checkPointGap int64 = 120
)

// Check handle trigger and last check and write new state of trigger, if state were change then write new NotificationEvent
func (triggerChecker *TriggerChecker) Check() error {
	triggerChecker.Logger.Debugf("Checking trigger %s", triggerChecker.TriggerID)
	checkData, err := triggerChecker.handleMetricsCheck()

	checkData, err = triggerChecker.handleTriggerCheck(checkData, err)
	if err != nil {
		return err
	}

	checkData.UpdateScore()
	return triggerChecker.Database.SetTriggerLastCheck(triggerChecker.TriggerID, &checkData, triggerChecker.trigger.IsRemote)
}

func (triggerChecker *TriggerChecker) handleMetricsCheck() (moira.CheckData, error) {
	checkData := copyLastCheck(triggerChecker.lastCheck, triggerChecker.Until)
	triggerMetricsData, metrics, err := triggerChecker.getFetchResult()
	if err != nil {
		return checkData, err
	}
	triggerChecker.cleanupMetricsValues(metrics, triggerChecker.Until)

	if len(triggerMetricsData.Main) == 0 {
		return checkData, ErrTriggerHasNoTimeSeries{}
	}

	if triggerMetricsData.HasOnlyWildcards() {
		return checkData, ErrTriggerHasOnlyWildcards{}
	}
	return triggerChecker.checkTriggerTimeSeries(triggerMetricsData, checkData)
}

func copyLastCheck(lastCheck *moira.CheckData, checkTimeStamp int64) moira.CheckData {
	lastMetrics := make(map[string]moira.MetricState, len(lastCheck.Metrics))
	for k, v := range lastCheck.Metrics {
		lastMetrics[k] = v
	}
	return moira.CheckData{
		Metrics:                      lastMetrics,
		State:                        lastCheck.State,
		Timestamp:                    checkTimeStamp,
		EventTimestamp:               lastCheck.EventTimestamp,
		Score:                        lastCheck.Score,
		Suppressed:                   lastCheck.Suppressed,
		SuppressedState:              lastCheck.SuppressedState,
		LastSuccessfulCheckTimestamp: lastCheck.LastSuccessfulCheckTimestamp,
	}
}

func (triggerChecker *TriggerChecker) checkTriggerTimeSeries(triggerMetricsData *metricSource.TriggerMetricsData, checkData moira.CheckData) (moira.CheckData, error) {
	timeSeriesNamesHash := make(map[string]bool, len(triggerMetricsData.Main))
	duplicateNamesHash := make(map[string]bool)

	for _, metricData := range triggerMetricsData.Main {
		triggerChecker.Logger.Debugf("[TriggerID:%s] Checking metricData %s: %v", triggerChecker.TriggerID, metricData.Name, metricData.Values)
		triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checking interval: %v - %v (%vs), step: %v", triggerChecker.TriggerID, metricData.Name, metricData.StartTime, metricData.StopTime, metricData.StepTime, metricData.StopTime-metricData.StartTime)

		if _, ok := timeSeriesNamesHash[metricData.Name]; ok {
			duplicateNamesHash[metricData.Name] = true
			triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Trigger has same timeseries names", triggerChecker.TriggerID, metricData.Name)
			continue
		}
		timeSeriesNamesHash[metricData.Name] = true
		metricState, needToDeleteMetric, err := triggerChecker.checkTimeSeries(metricData, triggerMetricsData)
		if needToDeleteMetric {
			triggerChecker.Logger.Infof("[TriggerID:%s] Remove metric: '%s'", triggerChecker.TriggerID, metricData.Name)
			delete(checkData.Metrics, metricData.Name)
			err = triggerChecker.Database.RemovePatternsMetrics(triggerChecker.trigger.Patterns)
		} else {
			checkData.Metrics[metricData.Name] = metricState
		}
		if err != nil {
			return checkData, err
		}
	}
	if len(duplicateNamesHash) > 0 {
		names := make([]string, 0, len(duplicateNamesHash))
		for key := range duplicateNamesHash {
			names = append(names, key)
		}
		return checkData, ErrTriggerHasSameTimeSeriesNames{names: names}
	}
	return checkData, nil
}

func (triggerChecker *TriggerChecker) checkTimeSeries(metricData *metricSource.MetricData, triggerMetricsData *metricSource.TriggerMetricsData) (lastState moira.MetricState, needToDeleteMetric bool, err error) {
	lastState = triggerChecker.lastCheck.GetOrCreateMetricState(metricData.Name, metricData.StartTime-3600, triggerChecker.trigger.MuteNewMetrics)
	metricStates, err := triggerChecker.getTimeSeriesStepsStates(triggerMetricsData, metricData, lastState)
	if err != nil {
		return
	}
	for _, currentState := range metricStates {
		lastState, err = triggerChecker.compareMetricStates(metricData.Name, currentState, lastState)
		if err != nil {
			return
		}
	}
	needToDeleteMetric, noDataState := triggerChecker.checkForNoData(metricData, lastState)
	if needToDeleteMetric {
		return
	}
	if noDataState != nil {
		lastState, err = triggerChecker.compareMetricStates(metricData.Name, *noDataState, lastState)
	}
	return
}

func (triggerChecker *TriggerChecker) handleTriggerCheck(checkData moira.CheckData, checkingError error) (moira.CheckData, error) {
	if checkingError == nil {
		checkData.State = OK
		if checkData.LastSuccessfulCheckTimestamp == 0 {
			checkData.LastSuccessfulCheckTimestamp = checkData.Timestamp
			return checkData, nil
		}
		checkData.LastSuccessfulCheckTimestamp = checkData.Timestamp
		return triggerChecker.compareTriggerStates(checkData)
	}

	switch checkingError.(type) {
	case ErrTriggerHasNoTimeSeries, ErrTriggerHasOnlyWildcards:
		triggerChecker.Logger.Debugf("Trigger %s: %s", triggerChecker.TriggerID, checkingError.Error())
		triggerState := ToTriggerState(triggerChecker.ttlState)
		if len(checkData.Metrics) == 0 {
			checkData.State = triggerState
			checkData.Message = checkingError.Error()
			if triggerChecker.ttl == 0 {
				return checkData, nil
			}
		}
	case ErrWrongTriggerTargets, ErrTriggerHasSameTimeSeriesNames:
		checkData.State = ERROR
		checkData.Message = checkingError.Error()
	case remote.ErrRemoteTriggerResponse:
		timeSinceLastSuccessfulCheck := checkData.Timestamp - checkData.LastSuccessfulCheckTimestamp
		if timeSinceLastSuccessfulCheck >= triggerChecker.ttl {
			checkData.State = EXCEPTION
			checkData.Message = fmt.Sprintf("Remote server unavailable. Trigger is not checked for %d seconds", timeSinceLastSuccessfulCheck)
		}
		triggerChecker.Logger.Errorf("Trigger %s: %s", triggerChecker.TriggerID, checkingError.Error())
	case local.ErrUnknownFunction, local.ErrEvalExpr:
		checkData.State = EXCEPTION
		checkData.Message = checkingError.Error()
		triggerChecker.Logger.Warningf("Trigger %s: %s", triggerChecker.TriggerID, checkingError.Error())
	default:
		triggerChecker.Metrics.CheckError.Mark(1)
		triggerChecker.Logger.Errorf("Trigger %s check failed: %s", triggerChecker.TriggerID, checkingError.Error())
	}
	return triggerChecker.compareTriggerStates(checkData)
}

func (triggerChecker *TriggerChecker) checkForNoData(metricData *metricSource.MetricData, metricLastState moira.MetricState) (bool, *moira.MetricState) {
	if triggerChecker.ttl == 0 {
		return false, nil
	}
	lastCheckTimeStamp := triggerChecker.lastCheck.Timestamp

	if metricLastState.Timestamp+triggerChecker.ttl >= lastCheckTimeStamp {
		return false, nil
	}
	triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Metric TTL expired for state %v", triggerChecker.TriggerID, metricData.Name, metricLastState)
	if triggerChecker.ttlState == DEL && metricLastState.EventTimestamp != 0 {
		return true, nil
	}
	return false, &moira.MetricState{
		State:       toMetricState(triggerChecker.ttlState),
		Timestamp:   lastCheckTimeStamp - triggerChecker.ttl,
		Value:       nil,
		Maintenance: metricLastState.Maintenance,
		Suppressed:  metricLastState.Suppressed,
	}
}

func (triggerChecker *TriggerChecker) getTimeSeriesStepsStates(triggerMetricsData *metricSource.TriggerMetricsData, metricData *metricSource.MetricData, metricLastState moira.MetricState) ([]moira.MetricState, error) {
	startTime := metricData.StartTime
	stepTime := metricData.StepTime

	checkPoint := metricLastState.GetCheckPoint(checkPointGap)
	triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checkpoint: %v", triggerChecker.TriggerID, metricData.Name, checkPoint)

	metricStates := make([]moira.MetricState, 0)

	for valueTimestamp := startTime; valueTimestamp < triggerChecker.Until+stepTime; valueTimestamp += stepTime {
		metricNewState, err := triggerChecker.getTimeSeriesState(triggerMetricsData, metricData, metricLastState, valueTimestamp, checkPoint)
		if err != nil {
			return nil, err
		}
		if metricNewState == nil {
			continue
		}
		metricLastState = *metricNewState
		metricStates = append(metricStates, *metricNewState)
	}
	return metricStates, nil
}

func (triggerChecker *TriggerChecker) getTimeSeriesState(triggerMetricsData *metricSource.TriggerMetricsData, metricData *metricSource.MetricData, lastState moira.MetricState, valueTimestamp, checkPoint int64) (*moira.MetricState, error) {
	if valueTimestamp <= checkPoint {
		return nil, nil
	}
	triggerExpression, noEmptyValues := getExpressionValues(triggerMetricsData, metricData, valueTimestamp)
	if !noEmptyValues {
		return nil, nil
	}
	triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Values for ts %v: MainTargetValue: %v, additionalTargetValues: %v", triggerChecker.TriggerID, metricData.Name, valueTimestamp, triggerExpression.MainTargetValue, triggerExpression.AdditionalTargetsValues)

	triggerExpression.WarnValue = triggerChecker.trigger.WarnValue
	triggerExpression.ErrorValue = triggerChecker.trigger.ErrorValue
	triggerExpression.TriggerType = triggerChecker.trigger.TriggerType
	triggerExpression.PreviousState = lastState.State
	triggerExpression.Expression = triggerChecker.trigger.Expression

	expressionState, err := triggerExpression.Evaluate()
	if err != nil {
		return nil, err
	}

	return &moira.MetricState{
		State:       expressionState,
		Timestamp:   valueTimestamp,
		Value:       &triggerExpression.MainTargetValue,
		Maintenance: lastState.Maintenance,
		Suppressed:  lastState.Suppressed,
	}, nil
}

func (triggerChecker *TriggerChecker) cleanupMetricsValues(metrics []string, until int64) {
	if len(metrics) > 0 {
		if err := triggerChecker.Database.RemoveMetricsValues(metrics, until-triggerChecker.Config.MetricsTTLSeconds); err != nil {
			triggerChecker.Logger.Error(err.Error())
		}
	}
}
