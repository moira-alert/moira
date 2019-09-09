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
	checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
	triggerMetricsData, err := triggerChecker.fetchTriggerMetrics()
	if err != nil {
		return checkData, err
	}
	return triggerChecker.checkTriggerMetrics(triggerMetricsData, checkData)
}

// Set new last check timestamp that equal to "until" targets fetch interval
// Do not copy message, if will be set if needed
func newCheckData(lastCheck *moira.CheckData, checkTimeStamp int64) moira.CheckData {
	lastMetrics := make(map[string]moira.MetricState, len(lastCheck.Metrics))
	for k, v := range lastCheck.Metrics {
		lastMetrics[k] = v
	}
	newCheckData := *lastCheck
	newCheckData.Metrics = lastMetrics
	newCheckData.Timestamp = checkTimeStamp
	newCheckData.Message = ""
	return newCheckData
}

func newMetricState(oldMetricState moira.MetricState, newState moira.State, newTimestamp int64, newValue *float64) *moira.MetricState {
	newMetricState := oldMetricState

	// This field always changed in every metric check operation
	newMetricState.State = newState
	newMetricState.Timestamp = newTimestamp
	newMetricState.Value = newValue

	// Always set. This fields only changed by user actions
	newMetricState.Maintenance = oldMetricState.Maintenance
	newMetricState.MaintenanceInfo = oldMetricState.MaintenanceInfo

	// Only can be change while understand that metric in maintenance or not in compareMetricStates logic
	newMetricState.Suppressed = oldMetricState.Suppressed

	// This fields always set in compareMetricStates logic
	// TODO: make sure that this logic can be moved here
	newMetricState.EventTimestamp = 0
	newMetricState.SuppressedState = ""
	return &newMetricState
}

func (triggerChecker *TriggerChecker) checkTriggerMetrics(triggerMetricsData *metricSource.TriggerMetricsData, checkData moira.CheckData) (moira.CheckData, error) {
	metricsDataToCheck, duplicateError := triggerChecker.getMetricsToCheck(triggerMetricsData.Main)

	for _, metricData := range metricsDataToCheck {
		triggerChecker.logger.Debugf("[TriggerID:%s] Checking metricData %s: %v", triggerChecker.triggerID, metricData.Name, metricData.Values)
		triggerChecker.logger.Debugf("[TriggerID:%s][MetricName:%s] Checking interval: %v - %v (%vs), step: %v", triggerChecker.triggerID, metricData.Name, metricData.StartTime, metricData.StopTime, metricData.StepTime, metricData.StopTime-metricData.StartTime)

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
		if metricData.Wildcard {
			continue
		}
		if _, ok := metricNamesHash[metricData.Name]; ok {
			triggerChecker.logger.Debugf("[TriggerID:%s][MetricName:%s] Trigger has same metric names", triggerChecker.triggerID, metricData.Name)
			duplicateNames = append(duplicateNames, metricData.Name)
			continue
		}
		metricNamesHash[metricData.Name] = struct{}{}
		metricsToCheck = append(metricsToCheck, metricData)
		delete(lastCheckMetricNamesHash, metricData.Name)
	}

	for metricName := range lastCheckMetricNamesHash {
		step := int64(60)
		if len(fetchedMetrics) > 0 && fetchedMetrics[0].StepTime != 0 {
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
		checkData.State = moira.StateOK
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
		triggerState := triggerChecker.ttlState.ToTriggerState()
		checkData.State = triggerState
		checkData.Message = checkingError.Error()
		if triggerChecker.ttl == 0 {
			// Do not alert when user don't wanna receive
			// NODATA state alerts, but change trigger status
			return checkData, nil
		}
	case ErrWrongTriggerTargets, ErrTriggerHasSameMetricNames:
		checkData.State = moira.StateERROR
		checkData.Message = checkingError.Error()
	case remote.ErrRemoteTriggerResponse:
		timeSinceLastSuccessfulCheck := checkData.Timestamp - checkData.LastSuccessfulCheckTimestamp
		if timeSinceLastSuccessfulCheck >= triggerChecker.ttl {
			checkData.State = moira.StateEXCEPTION
			checkData.Message = fmt.Sprintf("Remote server unavailable. Trigger is not checked for %d seconds", timeSinceLastSuccessfulCheck)
		}
		triggerChecker.logger.Errorf("Trigger %s: %s", triggerChecker.triggerID, checkingError.Error())
	case local.ErrUnknownFunction, local.ErrEvalExpr:
		checkData.State = moira.StateEXCEPTION
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
	triggerChecker.logger.Debugf("[TriggerID:%s][MetricName:%s] Metric TTL expired for state %v", triggerChecker.triggerID, metricData.Name, metricLastState)
	if triggerChecker.ttlState == moira.TTLStateDEL && metricLastState.EventTimestamp != 0 {
		return true, nil
	}
	return false, newMetricState(
		metricLastState,
		triggerChecker.ttlState.ToMetricState(),
		lastCheckTimeStamp,
		nil,
	)
}

func (triggerChecker *TriggerChecker) getMetricStepsStates(triggerMetricsData *metricSource.TriggerMetricsData, metricData *metricSource.MetricData, metricLastState moira.MetricState) ([]moira.MetricState, error) {
	startTime := metricData.StartTime
	stepTime := metricData.StepTime

	checkPoint := metricLastState.GetCheckPoint(checkPointGap)
	triggerChecker.logger.Debugf("[TriggerID:%s][MetricName:%s] Checkpoint: %v", triggerChecker.triggerID, metricData.Name, checkPoint)

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
	triggerChecker.logger.Debugf("[TriggerID:%s][MetricName:%s] Values for ts %v: MainTargetValue: %v, additionalTargetValues: %v", triggerChecker.triggerID, metricData.Name, valueTimestamp, triggerExpression.MainTargetValue, triggerExpression.AdditionalTargetsValues)

	triggerExpression.WarnValue = triggerChecker.trigger.WarnValue
	triggerExpression.ErrorValue = triggerChecker.trigger.ErrorValue
	triggerExpression.TriggerType = triggerChecker.trigger.TriggerType
	triggerExpression.PreviousState = lastState.State
	triggerExpression.Expression = triggerChecker.trigger.Expression

	expressionState, err := triggerExpression.Evaluate()
	if err != nil {
		return nil, err
	}

	return newMetricState(
		lastState,
		expressionState,
		valueTimestamp,
		&triggerExpression.MainTargetValue,
	), nil
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
