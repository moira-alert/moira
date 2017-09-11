package checker

import (
	"fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/target"
)

var checkPointGap int64 = 120

// ErrTriggerHasNoMetrics used if trigger no metrics
var ErrTriggerHasNoMetrics = fmt.Errorf("Trigger has no metrics")

// Check handle trigger and last check and write new state of trigger, if state were change then write new NotificationEvent
func (triggerChecker *TriggerChecker) Check() error {
	triggerChecker.Logger.Debugf("Checking trigger %s", triggerChecker.TriggerID)
	checkData, err := triggerChecker.handleTrigger()
	if err != nil {
		if err == ErrTriggerHasNoMetrics {
			triggerChecker.Logger.Warningf("Trigger %s: %s", triggerChecker.TriggerID, err.Error())
			checkData.State = triggerChecker.ttlState
			checkData.Message = ErrTriggerHasNoMetrics.Error()
		} else if target.IsErrUnknownFunction(err) {
			triggerChecker.Logger.Warningf("Trigger %s: %s", triggerChecker.TriggerID, err.Error())
			checkData.State = EXCEPTION
			checkData.Message = err.Error()
		} else {
			triggerChecker.Metrics.CheckError.Mark(1)
			triggerChecker.Logger.Errorf("Trigger %s check failed: %v", triggerChecker.TriggerID, err)
			checkData.State = EXCEPTION
			checkData.Message = "Trigger evaluation exception"
		}
		checkData, err = triggerChecker.compareChecks(checkData)
		if err != nil {
			return err
		}
	}

	checkData.Score = scores[checkData.State]
	for _, metricData := range checkData.Metrics {
		checkData.Score += scores[metricData.State]
	}
	return triggerChecker.Database.SetTriggerLastCheck(triggerChecker.TriggerID, &checkData)
}

func (triggerChecker *TriggerChecker) handleTrigger() (moira.CheckData, error) {
	lastMetrics := make(map[string]moira.MetricState)
	for k, v := range triggerChecker.lastCheck.Metrics {
		lastMetrics[k] = v
	}
	checkData := moira.CheckData{
		Metrics:   lastMetrics,
		State:     OK,
		Timestamp: triggerChecker.Until,
		Score:     triggerChecker.lastCheck.Score,
	}

	triggerTimeSeries, metrics, err := triggerChecker.getTimeSeries(triggerChecker.From, triggerChecker.Until)
	if err != nil {
		return checkData, err
	}

	triggerChecker.cleanupMetricsValues(metrics, triggerChecker.Until)

	hasMetrics, sendEvent := triggerChecker.hasMetrics(triggerTimeSeries)
	if !hasMetrics {
		if sendEvent {
			err = ErrTriggerHasNoMetrics
		}
		return checkData, err
	}

	for _, timeSeries := range triggerTimeSeries.Main {
		triggerChecker.Logger.Debugf("[TriggerID:%s] Checking timeSeries %s: %v", triggerChecker.TriggerID, timeSeries.Name, timeSeries.Values)
		triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checking interval: %v - %v (%vs), step: %v", triggerChecker.TriggerID, timeSeries.Name, timeSeries.StartTime, timeSeries.StopTime, timeSeries.StepTime, timeSeries.StopTime-timeSeries.StartTime)

		metricLastState := triggerChecker.lastCheck.GetOrCreateMetricState(timeSeries.Name, int64(timeSeries.StartTime-3600))
		metricStates, err := triggerChecker.getTimeSeriesStepsStates(triggerTimeSeries, timeSeries, metricLastState)
		if err != nil {
			return checkData, nil
		}
		for _, metricState := range metricStates {
			currentState, err := triggerChecker.compareStates(timeSeries.Name, metricState, metricLastState)
			metricLastState = currentState
			checkData.Metrics[timeSeries.Name] = currentState
			if err != nil {
				return checkData, err
			}
		}

		needToDeleteMetric, currentState := triggerChecker.checkForNoData(timeSeries, metricLastState)
		if needToDeleteMetric {
			triggerChecker.Logger.Infof("[TriggerID:%s] Remove metric: '%s'", triggerChecker.TriggerID, timeSeries.Name)
			delete(checkData.Metrics, timeSeries.Name)
			if err := triggerChecker.Database.RemovePatternsMetrics(triggerChecker.trigger.Patterns); err != nil {
				return checkData, err
			}
			continue
		}
		if currentState != nil {
			currentState, err := triggerChecker.compareStates(timeSeries.Name, *currentState, metricLastState)
			checkData.Metrics[timeSeries.Name] = currentState
			if err != nil {
				return checkData, err
			}
		}
	}
	return checkData, nil
}

func (triggerChecker *TriggerChecker) hasMetrics(tts *triggerTimeSeries) (hasMetrics, sendEvent bool) {
	hasMetrics = true
	sendEvent = false

	if len(tts.Main) == 0 {
		hasMetrics = false
		if triggerChecker.ttl != 0 && len(triggerChecker.lastCheck.Metrics) != 0 {
			sendEvent = true
		}
	}
	return hasMetrics, sendEvent
}

func (triggerChecker *TriggerChecker) checkForNoData(timeSeries *target.TimeSeries, metricLastState moira.MetricState) (bool, *moira.MetricState) {
	if triggerChecker.ttl == 0 {
		return false, nil
	}
	lastCheckTimeStamp := triggerChecker.lastCheck.Timestamp

	if metricLastState.Timestamp+triggerChecker.ttl >= lastCheckTimeStamp {
		return false, nil
	}
	triggerChecker.Logger.Infof("[TriggerID:%s][TimeSeries:%s] Metric TTL expired for state %v", triggerChecker.TriggerID, timeSeries.Name, metricLastState)
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

func (triggerChecker *TriggerChecker) getTimeSeriesStepsStates(triggerTimeSeries *triggerTimeSeries, timeSeries *target.TimeSeries, metricLastState moira.MetricState) ([]moira.MetricState, error) {
	startTime := int64(timeSeries.StartTime)
	stepTime := int64(timeSeries.StepTime)

	checkPoint := metricLastState.GetCheckPoint(checkPointGap)
	triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checkpoint: %v", triggerChecker.TriggerID, timeSeries.Name, checkPoint)

	metricStates := make([]moira.MetricState, 0)

	for valueTimestamp := startTime; valueTimestamp < triggerChecker.Until+stepTime; valueTimestamp += stepTime {
		metricNewState, err := triggerChecker.getTimeSeriesState(triggerTimeSeries, timeSeries, metricLastState, valueTimestamp, checkPoint)
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

func (triggerChecker *TriggerChecker) getTimeSeriesState(triggerTimeSeries *triggerTimeSeries, timeSeries *target.TimeSeries, lastState moira.MetricState, valueTimestamp, checkPoint int64) (*moira.MetricState, error) {
	if valueTimestamp <= checkPoint {
		return nil, nil
	}
	triggerExpression, noEmptyValues := triggerTimeSeries.getExpressionValues(timeSeries, valueTimestamp)
	if !noEmptyValues {
		return nil, nil
	}
	triggerChecker.Logger.Debugf("[TriggerID:%s][TimeSeries:%s] Values for ts %v: MainTargetValue: %v, additionalTargetValues: %v", triggerChecker.TriggerID, timeSeries.Name, valueTimestamp, triggerExpression.MainTargetValue, triggerExpression.AdditionalTargetsValues)

	triggerExpression.WarnValue = triggerChecker.trigger.WarnValue
	triggerExpression.ErrorValue = triggerChecker.trigger.ErrorValue
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
	for _, metric := range metrics {
		if err := triggerChecker.Database.RemoveMetricValues(metric, until-triggerChecker.Config.MetricsTTL); err != nil {
			triggerChecker.Logger.Error(err.Error())
		}
	}
}
