package checker

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"
)

var (
	checkPointGap int64 = 120
)

// Check handle trigger and last check and write new state of trigger, if state were change then write new NotificationEvent
func (triggerChecker *TriggerChecker) Check() error {
	triggerChecker.logger.Debugf("Checking trigger %s", triggerChecker.triggerID)
	checkData, err := triggerChecker.checkTrigger()

	checkData, err = triggerChecker.handleCheckResult(checkData, err)
	if err != nil {
		return err
	}

	checkData.UpdateScore()
	return triggerChecker.database.SetTriggerLastCheck(triggerChecker.triggerID, &checkData, triggerChecker.trigger.IsRemote)
}

func (triggerChecker *TriggerChecker) checkTrigger() (moira.CheckData, error) {
	checkData := copyLastCheck(triggerChecker.lastCheck, triggerChecker.until)
	triggerMetricsData, err := triggerChecker.fetchTriggerMetrics()
	if err != nil {
		return checkData, err
	}
	return triggerChecker.checkTriggerMetrics(triggerMetricsData, checkData)
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
		Maintenance:                  lastCheck.Maintenance,
		LastSuccessfulCheckTimestamp: lastCheck.LastSuccessfulCheckTimestamp,
	}
}

func (triggerChecker *TriggerChecker) checkTriggerMetrics(triggerMetricsData *metricSource.TriggerMetricsData, checkData moira.CheckData) (moira.CheckData, error) {
	metricsDataToCheck, duplicateError := triggerChecker.getMetricsToCheck(triggerMetricsData.Main)

	for _, metricData := range metricsDataToCheck {
		triggerChecker.logger.Debugf("[TriggerID:%s] Checking metricData %s: %v", triggerChecker.triggerID, metricData.Name, metricData.Values)
		triggerChecker.logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checking interval: %v - %v (%vs), step: %v", triggerChecker.triggerID, metricData.Name, metricData.StartTime, metricData.StopTime, metricData.StepTime, metricData.StopTime-metricData.StartTime)

		metricState, needToDeleteMetric, err := triggerChecker.checkMetricData(metricData, triggerMetricsData)
		if needToDeleteMetric {
			triggerChecker.logger.Infof("[TriggerID:%s] Remove metric: '%s'", triggerChecker.triggerID, metricData.Name)
			delete(checkData.Metrics, metricData.Name)
			err = triggerChecker.database.RemovePatternsMetrics(triggerChecker.trigger.Patterns)
		} else {
			checkData.Metrics[metricData.Name] = metricState
		}
		if err != nil {
			return checkData, err
		}
	}
	return checkData, duplicateError
}

func (triggerChecker *TriggerChecker) getMetricsToCheck(fetchedMetrics []*metricSource.MetricData) ([]*metricSource.MetricData, error) {
	metricNamesHash := make(map[string]struct{}, len(fetchedMetrics))
	duplicateNames := make([]string, 0)
	metricsToCheck := make([]*metricSource.MetricData, 0, len(fetchedMetrics))
	lastCheckMetricNamesHash := make(map[string]struct{}, len(triggerChecker.lastCheck.Metrics))
	for metricName := range triggerChecker.lastCheck.Metrics {
		lastCheckMetricNamesHash[metricName] = struct{}{}
	}

	for _, metricData := range fetchedMetrics {
		if _, ok := metricNamesHash[metricData.Name]; ok {
			triggerChecker.logger.Debugf("[TriggerID:%s][TimeSeries:%s] Trigger has same timeseries names", triggerChecker.triggerID, metricData.Name)
			duplicateNames = append(duplicateNames, metricData.Name)
			continue
		}
		metricNamesHash[metricData.Name] = struct{}{}
		metricsToCheck = append(metricsToCheck, metricData)
		delete(lastCheckMetricNamesHash, metricData.Name)
	}

	for metricName := range lastCheckMetricNamesHash {
		step := int64(60)
		if len(fetchedMetrics) > 0 {
			step = fetchedMetrics[0].StepTime
		}
		metricData := metricSource.MakeEmptyMetricData(metricName, step, triggerChecker.from, triggerChecker.until)
		metricsToCheck = append(metricsToCheck, metricData)
	}

	if len(duplicateNames) > 0 {
		return metricsToCheck, ErrTriggerHasSameMetricNames{names: duplicateNames}
	}
	return metricsToCheck, nil
}

func (triggerChecker *TriggerChecker) checkMetricData(metricData *metricSource.MetricData, triggerMetricsData *metricSource.TriggerMetricsData) (lastState moira.MetricState, needToDeleteMetric bool, err error) {
	lastState = triggerChecker.lastCheck.GetOrCreateMetricState(metricData.Name, metricData.StartTime-3600, triggerChecker.trigger.MuteNewMetrics)
	metricStates, err := triggerChecker.getMetricStepsStates(triggerMetricsData, metricData, lastState)
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

func (triggerChecker *TriggerChecker) handleCheckResult(checkData moira.CheckData, checkingError error) (moira.CheckData, error) {
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
	case ErrTriggerHasNoMetrics, ErrTriggerHasOnlyWildcards:
		triggerChecker.logger.Debugf("Trigger %s: %s", triggerChecker.triggerID, checkingError.Error())
		triggerState := ToTriggerState(triggerChecker.ttlState)
		if len(checkData.Metrics) == 0 {
			checkData.State = triggerState
			checkData.Message = checkingError.Error()
			if triggerChecker.ttl == 0 {
				return checkData, nil
			}
		}
	case ErrWrongTriggerTargets, ErrTriggerHasSameMetricNames:
		checkData.State = ERROR
		checkData.Message = checkingError.Error()
	case remote.ErrRemoteTriggerResponse:
		timeSinceLastSuccessfulCheck := checkData.Timestamp - checkData.LastSuccessfulCheckTimestamp
		if timeSinceLastSuccessfulCheck >= triggerChecker.ttl {
			checkData.State = EXCEPTION
			checkData.Message = fmt.Sprintf("Remote server unavailable. Trigger is not checked for %d seconds", timeSinceLastSuccessfulCheck)
		}
		triggerChecker.logger.Errorf("Trigger %s: %s", triggerChecker.triggerID, checkingError.Error())
	case local.ErrUnknownFunction, local.ErrEvalExpr:
		checkData.State = EXCEPTION
		checkData.Message = checkingError.Error()
		triggerChecker.logger.Warningf("Trigger %s: %s", triggerChecker.triggerID, checkingError.Error())
	default:
		triggerChecker.metrics.CheckError.Mark(1)
		triggerChecker.logger.Errorf("Trigger %s check failed: %s", triggerChecker.triggerID, checkingError.Error())
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
	triggerChecker.logger.Debugf("[TriggerID:%s][TimeSeries:%s] Metric TTL expired for state %v", triggerChecker.triggerID, metricData.Name, metricLastState)
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

func (triggerChecker *TriggerChecker) getMetricStepsStates(triggerMetricsData *metricSource.TriggerMetricsData, metricData *metricSource.MetricData, metricLastState moira.MetricState) ([]moira.MetricState, error) {
	startTime := metricData.StartTime
	stepTime := metricData.StepTime

	checkPoint := metricLastState.GetCheckPoint(checkPointGap)
	triggerChecker.logger.Debugf("[TriggerID:%s][TimeSeries:%s] Checkpoint: %v", triggerChecker.triggerID, metricData.Name, checkPoint)

	metricStates := make([]moira.MetricState, 0)

	for valueTimestamp := startTime; valueTimestamp < triggerChecker.until+stepTime; valueTimestamp += stepTime {
		metricNewState, err := triggerChecker.getMetricDataState(triggerMetricsData, metricData, metricLastState, valueTimestamp, checkPoint)
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

func (triggerChecker *TriggerChecker) getMetricDataState(triggerMetricsData *metricSource.TriggerMetricsData, metricData *metricSource.MetricData, lastState moira.MetricState, valueTimestamp, checkPoint int64) (*moira.MetricState, error) {
	if valueTimestamp <= checkPoint {
		return nil, nil
	}
	triggerExpression, noEmptyValues := getExpressionValues(triggerMetricsData, metricData, valueTimestamp)
	if !noEmptyValues {
		return nil, nil
	}
	triggerChecker.logger.Debugf("[TriggerID:%s][TimeSeries:%s] Values for ts %v: MainTargetValue: %v, additionalTargetValues: %v", triggerChecker.triggerID, metricData.Name, valueTimestamp, triggerExpression.MainTargetValue, triggerExpression.AdditionalTargetsValues)

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

func getExpressionValues(triggerMetricsData *metricSource.TriggerMetricsData, firstTargetMetricData *metricSource.MetricData, valueTimestamp int64) (*expression.TriggerExpression, bool) {
	expressionValues := &expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64, len(triggerMetricsData.Additional)),
	}
	firstTargetValue := firstTargetMetricData.GetTimestampValue(valueTimestamp)
	if !moira.IsValidFloat64(firstTargetValue) {
		return expressionValues, false
	}
	expressionValues.MainTargetValue = firstTargetValue

	for targetNumber := 0; targetNumber < len(triggerMetricsData.Additional); targetNumber++ {
		additionalMetricData := triggerMetricsData.Additional[targetNumber]
		if additionalMetricData == nil {
			return expressionValues, false
		}
		tnValue := additionalMetricData.GetTimestampValue(valueTimestamp)
		if !moira.IsValidFloat64(tnValue) {
			return expressionValues, false
		}
		expressionValues.AdditionalTargetsValues[triggerMetricsData.GetAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}
