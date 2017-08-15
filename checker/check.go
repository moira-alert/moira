package checker

import (
	"github.com/go-errors/errors"
	"github.com/moira-alert/moira-alert"
	"math"
)

var checkPointGap int64 = 120

var ErrTriggerHasNoMetrics = errors.New("Trigger has no metrics")

func (triggerChecker *TriggerChecker) Check() error {
	checkData, err := triggerChecker.handleTrigger()
	if err != nil {
		if err == ErrTriggerHasNoMetrics {
			checkData.State = triggerChecker.ttlState
			checkData.Message = "Trigger has no metrics"
		} else {
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
	return triggerChecker.Database.SetTriggerLastCheck(triggerChecker.TriggerId, &checkData)
}

func (triggerChecker *TriggerChecker) handleTrigger() (moira.CheckData, error) {
	checkData := moira.CheckData{
		Metrics:   triggerChecker.lastCheck.Metrics,
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
		triggerChecker.Logger.Debugf("Checking timeSeries %s: %v", timeSeries.Name, timeSeries.Values)
		triggerChecker.Logger.Debugf("Checking interval: %v - %v (%vs), step: %v", timeSeries.StartTime, timeSeries.StopTime, timeSeries.StepTime, timeSeries.StopTime-timeSeries.StartTime)
		metricLastState := triggerChecker.lastCheck.GetOrCreateMetricState(timeSeries.Name, int64(timeSeries.StartTime-3600))

		metricStates := triggerChecker.getTimeSeriesStepsStates(triggerTimeSeries, timeSeries, metricLastState)
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
			triggerChecker.Logger.Infof("Remove metric %s", timeSeries.Name)
			delete(checkData.Metrics, timeSeries.Name)
			if err := triggerChecker.Database.RemovePatternsMetrics(triggerChecker.trigger.Patterns); err != nil {
				return checkData, err
			}
			continue
		}
		if currentState != nil {
			currentState, err := triggerChecker.compareStates(timeSeries.Name, *currentState, metricLastState)
			metricLastState = currentState
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
		if triggerChecker.ttl != nil && len(triggerChecker.lastCheck.Metrics) != 0 {
			sendEvent = true
		}
	}
	return hasMetrics, sendEvent
}

func (triggerChecker *TriggerChecker) checkForNoData(timeSeries *TimeSeries, metricLastState moira.MetricState) (bool, *moira.MetricState) {
	if triggerChecker.ttl == nil {
		return false, nil
	}
	lastCheckTimeStamp := triggerChecker.lastCheck.Timestamp
	ttl := *triggerChecker.ttl

	if metricLastState.Timestamp+ttl >= lastCheckTimeStamp {
		return false, nil
	}
	triggerChecker.Logger.Infof("Metric %s TTL expired for state %v", timeSeries.Name, metricLastState)
	if triggerChecker.ttlState == DEL && metricLastState.EventTimestamp != 0 {
		return true, nil
	}
	return false, &moira.MetricState{
		State:       toMetricState(triggerChecker.ttlState),
		Timestamp:   lastCheckTimeStamp - ttl,
		Value:       nil,
		Maintenance: metricLastState.Maintenance,
		Suppressed:  metricLastState.Suppressed,
	}
}

func (triggerChecker *TriggerChecker) getTimeSeriesStepsStates(triggerTimeSeries *triggerTimeSeries, timeSeries *TimeSeries, metricLastState moira.MetricState) []moira.MetricState {
	startTime := int64(timeSeries.StartTime)
	stepTime := int64(timeSeries.StepTime)

	checkPoint := metricLastState.GetCheckPoint(checkPointGap)
	triggerChecker.Logger.Debugf("Checkpoint for %s: %v", timeSeries.Name, checkPoint)

	metricStates := make([]moira.MetricState, 0)

	for valueTimestamp := startTime; valueTimestamp < triggerChecker.Until+stepTime; valueTimestamp += stepTime {
		metricNewState := triggerChecker.getTimeSeriesState(triggerTimeSeries, timeSeries, metricLastState, valueTimestamp, checkPoint)
		if metricNewState == nil {
			continue
		}
		metricStates = append(metricStates, *metricNewState)
	}
	return metricStates
}

func (triggerChecker *TriggerChecker) getTimeSeriesState(triggerTimeSeries *triggerTimeSeries, timeSeries *TimeSeries, lastState moira.MetricState, valueTimestamp, checkPoint int64) *moira.MetricState {
	if valueTimestamp <= checkPoint {
		return nil
	}
	expressionValues, noEmptyValues := triggerTimeSeries.getExpressionValues(timeSeries, valueTimestamp)
	triggerChecker.Logger.Debugf("values for ts %v: %v", valueTimestamp, expressionValues)
	if !noEmptyValues {
		return nil
	}

	expressionValues["warn_value"] = getMathFloat64Pointer(triggerChecker.trigger.WarnValue)
	expressionValues["error_value"] = getMathFloat64Pointer(triggerChecker.trigger.ErrorValue)
	expressionValues["PREV_STATE"] = 1000 //todo NODATA

	expressionState := GetExpression(triggerChecker.trigger.Expression, expressionValues)

	value := expressionValues.GetTargetValue(triggerTimeSeries.getMainTargetName())

	return &moira.MetricState{
		State:       expressionState,
		Timestamp:   valueTimestamp,
		Value:       value,
		Maintenance: lastState.Maintenance,
		Suppressed:  lastState.Suppressed,
	}
}

func (triggerChecker *TriggerChecker) cleanupMetricsValues(metrics []string, until int64) {
	for _, metric := range metrics {
		go func(metric string) {
			//todo cache cache_ttl
			if err := triggerChecker.Database.CleanupMetricValues(metric, until-triggerChecker.Config.MetricsTTL); err != nil {
				triggerChecker.Logger.Error(err.Error())
			}
		}(metric)
	}
}

func getMathFloat64Pointer(val *float64) float64 {
	if val != nil {
		return *val
	} else {
		return math.NaN()
	}
}
