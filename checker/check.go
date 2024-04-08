package checker

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker/metrics/conversion"
	"github.com/moira-alert/moira/expression"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/moira-alert/moira/metric_source/remote"
)

const (
	checkPointGap int64 = 120
)

// Check handle trigger and last check and write new state of trigger, if state were change then write new NotificationEvent.
func (triggerChecker *TriggerChecker) Check() error {
	triggerChecker.logger.Debug().Msg("Checking trigger")

	checkData := newCheckData(triggerChecker.lastCheck, triggerChecker.until)
	errorSeverity := NoCheckError

	triggerMetricsData, err := triggerChecker.fetchTriggerMetrics()
	if err != nil {
		return triggerChecker.handleFetchError(checkData, err)
	}

	preparedMetrics, aloneMetrics, err := triggerChecker.prepareMetrics(triggerMetricsData)
	if err != nil {
		errorSeverity, checkData, err = triggerChecker.handlePrepareError(checkData, err)
		if errorSeverity == MustStopCheck {
			return err
		}
	}

	checkData.MetricsToTargetRelation = conversion.GetRelations(aloneMetrics, triggerChecker.trigger.AloneMetrics)

	checkData, err = triggerChecker.check(preparedMetrics, aloneMetrics, checkData, triggerChecker.logger)
	if err != nil {
		return triggerChecker.handleUndefinedError(checkData, err)
	}

	if errorSeverity == NoCheckError {
		checkData.State = moira.StateOK
	}

	checkData.LastSuccessfulCheckTimestamp = checkData.Timestamp
	if checkData.LastSuccessfulCheckTimestamp != 0 {
		checkData, err = triggerChecker.compareTriggerStates(checkData)
		if err != nil {
			return err
		}
	}

	checkData.UpdateScore()
	return triggerChecker.database.SetTriggerLastCheck(
		triggerChecker.triggerID,
		&checkData,
		triggerChecker.trigger.ClusterKey(),
	)
}

type ErrorSeverity int

const (
	NoCheckError     ErrorSeverity = 0
	CanContinueCheck ErrorSeverity = 1
	MustStopCheck    ErrorSeverity = 2
)

// handlePrepareError is a function that checks error returned from prepareMetrics function.
// If error is not serious and check process can be continued first return value became CanContinueCheck and Filled CheckData returned.
// In the other case first return value became MustStopCheck and error passed to this function is handled.
func (triggerChecker *TriggerChecker) handlePrepareError(checkData moira.CheckData, err error) (ErrorSeverity, moira.CheckData, error) {
	switch err.(type) { // nolint:errorlint
	case ErrTriggerHasSameMetricNames:
		checkData.State = moira.StateEXCEPTION
		checkData.Message = err.Error()
		return CanContinueCheck, checkData, nil

	case conversion.ErrUnexpectedAloneMetric:
		checkData.State = moira.StateEXCEPTION
		checkData.Message = err.Error()
		logTriggerCheckException(triggerChecker.logger, triggerChecker.triggerID, err)

	case conversion.ErrEmptyAloneMetricsTarget:
		checkData.State = moira.StateNODATA
		triggerChecker.logger.Warning().
			Error(err).
			Msg("Trigger check failed")

	default:
		return MustStopCheck, checkData, triggerChecker.handleUndefinedError(checkData, err)
	}

	checkData, err = triggerChecker.compareTriggerStates(checkData)
	if err != nil {
		return MustStopCheck, checkData, err
	}

	checkData.UpdateScore()
	err = triggerChecker.database.SetTriggerLastCheck(
		triggerChecker.triggerID,
		&checkData,
		triggerChecker.trigger.ClusterKey(),
	)

	return MustStopCheck, checkData, err
}

// handleFetchError is a function that checks error returned from fetchTriggerMetrics function.
func (triggerChecker *TriggerChecker) handleFetchError(checkData moira.CheckData, err error) error {
	switch err.(type) { // nolint:errorlint
	case ErrTriggerHasEmptyTargets, ErrTriggerHasOnlyWildcards:
		triggerChecker.logger.Debug().
			String(moira.LogFieldNameTriggerID, triggerChecker.triggerID).
			Error(err).
			Msg("Trigger was fetched")

		triggerState := triggerChecker.ttlState.ToTriggerState()
		checkData.State = triggerState
		checkData.Message = err.Error()
		if triggerChecker.ttl == 0 {
			// Do not alert when user don't wanna receive
			// NODATA state alerts, but change trigger status
			checkData.UpdateScore()
			return triggerChecker.database.SetTriggerLastCheck(
				triggerChecker.triggerID,
				&checkData,
				triggerChecker.trigger.ClusterKey(),
			)
		}
	case remote.ErrRemoteTriggerResponse:
		timeSinceLastSuccessfulCheck := checkData.Timestamp - checkData.LastSuccessfulCheckTimestamp
		if timeSinceLastSuccessfulCheck >= triggerChecker.ttl {
			checkData.State = moira.StateEXCEPTION
			checkData.Message = fmt.Sprintf("Remote server unavailable. Trigger is not checked for %d seconds", timeSinceLastSuccessfulCheck)
			checkData, err = triggerChecker.compareTriggerStates(checkData)
		}
		logTriggerCheckException(triggerChecker.logger, triggerChecker.triggerID, err)
	case local.ErrUnknownFunction, local.ErrEvalExpr:
		checkData.State = moira.StateEXCEPTION
		checkData.Message = err.Error()
		logTriggerCheckException(triggerChecker.logger, triggerChecker.triggerID, err)
	default:
		return triggerChecker.handleUndefinedError(checkData, err)
	}

	checkData, err = triggerChecker.compareTriggerStates(checkData)
	if err != nil {
		return err
	}

	checkData.UpdateScore()
	return triggerChecker.database.SetTriggerLastCheck(
		triggerChecker.triggerID,
		&checkData,
		triggerChecker.trigger.ClusterKey(),
	)
}

// handleUndefinedError is a function that check error with undefined type.
func (triggerChecker *TriggerChecker) handleUndefinedError(checkData moira.CheckData, err error) error {
	triggerChecker.metrics.CheckError.Mark(1)
	checkData.State = moira.StateEXCEPTION
	checkData.Message = err.Error()

	triggerChecker.logger.Error().
		String(moira.LogFieldNameTriggerID, triggerChecker.triggerID).
		Error(err).
		Msg("Trigger check failed")

	checkData, err = triggerChecker.compareTriggerStates(checkData)
	if err != nil {
		return err
	}

	checkData.UpdateScore()
	return triggerChecker.database.SetTriggerLastCheck(
		triggerChecker.triggerID,
		&checkData,
		triggerChecker.trigger.ClusterKey(),
	)
}

func logTriggerCheckException(logger moira.Logger, triggerID string, err error) {
	logger.Warning().
		Error(err).
		String(moira.LogFieldNameTriggerID, triggerID).
		Msg("Trigger check failed")
}

// Set new last check timestamp that equal to "until" targets fetch interval.
// Do not copy message, it will be set if needed.
func newCheckData(lastCheck *moira.CheckData, checkTimeStamp int64) moira.CheckData {
	lastMetrics := make(map[string]moira.MetricState, len(lastCheck.Metrics))
	for k, v := range lastCheck.Metrics {
		lastMetrics[k] = v
	}

	metricsToTargetRelation := make(map[string]string, len(lastCheck.MetricsToTargetRelation))
	for k, v := range lastCheck.MetricsToTargetRelation {
		metricsToTargetRelation[k] = v
	}

	newCheckData := *lastCheck
	newCheckData.Metrics = lastMetrics
	newCheckData.Timestamp = checkTimeStamp
	newCheckData.MetricsToTargetRelation = metricsToTargetRelation
	newCheckData.Message = ""

	return newCheckData
}

func newMetricState(oldMetricState moira.MetricState, newState moira.State, newTimestamp int64, newValues map[string]float64) *moira.MetricState {
	newMetricState := oldMetricState

	// This field always changed in every metric check operation
	newMetricState.State = newState
	newMetricState.Timestamp = newTimestamp
	newMetricState.Values = newValues

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

// prepareMetrics is a function that takes fetched metrics and prepare it to check.
// The sequence of check is following:
// Call preparePatternMetrics that converts fetched metrics to TriggerPatternMetrics ->.
// Populate metrics ->.
// Filter alone metrics ->.
// Check that targets with alone metrics declared in trigger ->.
// Convert to TriggerMetricsToCheck.
func (triggerChecker *TriggerChecker) prepareMetrics(fetchedMetrics map[string][]metricSource.MetricData) (map[string]map[string]metricSource.MetricData, map[string]metricSource.MetricData, error) {
	from := triggerChecker.from
	to := triggerChecker.until
	preparedPatternMetrics := conversion.NewTriggerMetricsWithCapacity(len(fetchedMetrics))
	duplicates := make(map[string][]string)

	for targetName, patternMetrics := range fetchedMetrics {
		prepared, patternDuplicates := triggerChecker.preparePatternMetrics(patternMetrics)
		preparedPatternMetrics[targetName] = prepared
		if len(patternDuplicates) > 0 {
			duplicates[targetName] = patternDuplicates
		}
	}

	multiMetricTargets, aloneMetrics, err := preparedPatternMetrics.FilterAloneMetrics(triggerChecker.trigger.AloneMetrics)
	if err != nil {
		return nil, nil, err
	}

	populatedAloneMetrics, err := aloneMetrics.Populate(triggerChecker.lastCheck.MetricsToTargetRelation, triggerChecker.trigger.AloneMetrics, from, to)
	if err != nil {
		return nil, nil, err
	}

	populated := multiMetricTargets.Populate(triggerChecker.lastCheck.Metrics, triggerChecker.trigger.AloneMetrics, from, to)

	converted := populated.ConvertForCheck()
	if len(duplicates) > 0 {
		return converted, populatedAloneMetrics, NewErrTriggerHasSameMetricNames(duplicates)
	}

	return converted, populatedAloneMetrics, nil
}

// preparePatternMetrics is a function that takes PatternMetrics and applies following operations on it:
// PatternMetrics ->.
// Remove wildcards ->.
// Remove duplicated metrics and collect the names of duplicated metrics ->.
// Convert to TriggerPatternMetrics.
func (triggerChecker *TriggerChecker) preparePatternMetrics(fetchedMetrics conversion.FetchedTargetMetrics) (conversion.TriggerTargetMetrics, []string) {
	withoutWildcards := fetchedMetrics.CleanWildcards()
	deduplicated, duplicates := withoutWildcards.Deduplicate()

	result := conversion.NewTriggerTargetMetrics(deduplicated)

	return result, duplicates
}

/*
check - function that calculates the state of metrics in the trigger

2 cases are possible:

1) The trigger has only alone targets, in this case a metric with the name of the first target
is created and then the values of other alone metrics are added to it

2) The trigger has regular metrics, in this case all alone metrics are simply added to
all regular metrics.
*/
func (triggerChecker *TriggerChecker) check(
	regularMetrics map[string]map[string]metricSource.MetricData,
	aloneMetrics map[string]metricSource.MetricData,
	checkData moira.CheckData,
	logger moira.Logger,
) (moira.CheckData, error) {
	// Case when trigger have only alone metrics
	if len(regularMetrics) == 0 {
		return triggerChecker.handleAloneMetrics(aloneMetrics, checkData, logger)
	}

	return triggerChecker.checkRegularMetrics(regularMetrics, aloneMetrics, checkData, logger)
}

func (triggerChecker *TriggerChecker) handleAloneMetrics(
	aloneMetrics map[string]metricSource.MetricData,
	checkData moira.CheckData,
	logger moira.Logger,
) (moira.CheckData, error) {
	// We should not create a metric if the number of alone metrics does not match
	// the number of targets, otherwise errors may occur during the calculation of
	// the expression
	if len(aloneMetrics) != len(triggerChecker.trigger.Targets) {
		return checkData, nil
	}

	// Create a regular metric with first alone metric name
	metricName := conversion.MetricName(aloneMetrics)
	regularMetrics := map[string]map[string]metricSource.MetricData{
		metricName: make(map[string]metricSource.MetricData),
	}

	return triggerChecker.checkRegularMetrics(regularMetrics, aloneMetrics, checkData, logger)
}

// Checks if the metric has changed since the previous check.
func isMetricChanged(metrics map[string]moira.MetricState, metricName string, metricState moira.MetricState) bool {
	prevMetricState := metrics[metricName]
	return prevMetricState.Timestamp != metricState.Timestamp
}

func (triggerChecker *TriggerChecker) checkRegularMetrics(
	regularMetrics map[string]map[string]metricSource.MetricData,
	aloneMetrics map[string]metricSource.MetricData,
	checkData moira.CheckData,
	logger moira.Logger,
) (moira.CheckData, error) {
	for metricName, targets := range regularMetrics {
		log := logger.Clone().
			String(moira.LogFieldNameMetricName, metricName)

		targets = conversion.Merge(targets, aloneMetrics)

		metricState, needToDeleteMetric, err := triggerChecker.checkTargets(metricName, targets, log)

		if needToDeleteMetric {
			log.Debug().Msg("Remove metric")

			checkData.RemoveMetricState(metricName)
			err = triggerChecker.database.RemovePatternsMetrics(triggerChecker.trigger.Patterns)
		} else {
			// Starting to show user the updated metric, which has been hidden as its Maintenance time is not over
			if metricState.DeletedButKept && isMetricChanged(checkData.Metrics, metricName, metricState) {
				metricState.DeletedButKept = false
			}
			checkData.Metrics[metricName] = metricState
		}

		if err != nil {
			return checkData, err
		}
	}

	return checkData, nil
}

// checkTargets is a function which determines the last state of the metric and information about whether it should be deleted.
func (triggerChecker *TriggerChecker) checkTargets(
	metricName string,
	metrics map[string]metricSource.MetricData,
	logger moira.Logger,
) (
	lastState moira.MetricState,
	needToDeleteMetric bool,
	err error,
) {
	lastState, metricStates, err := triggerChecker.getMetricStepsStates(metricName, metrics, logger)
	if err != nil {
		return lastState, needToDeleteMetric, err
	}

	for _, currentState := range metricStates {
		lastState, err = triggerChecker.compareMetricStates(metricName, currentState, lastState)
		if err != nil {
			return lastState, needToDeleteMetric, err
		}
	}

	needToDeleteMetric, noDataState := triggerChecker.checkForNoData(lastState, logger)
	if needToDeleteMetric {
		return lastState, needToDeleteMetric, err
	}

	if noDataState != nil {
		lastState, err = triggerChecker.compareMetricStates(metricName, *noDataState, lastState)
	}

	return lastState, needToDeleteMetric, err
}

func (triggerChecker *TriggerChecker) checkForNoData(
	metricLastState moira.MetricState,
	logger moira.Logger,
) (
	needToDeleteMetric bool,
	noDataState *moira.MetricState,
) {
	if triggerChecker.ttl == 0 {
		return false, nil
	}

	lastCheckTimeStamp := triggerChecker.lastCheck.Timestamp

	if metricLastState.Timestamp+triggerChecker.ttl >= lastCheckTimeStamp {
		return false, nil
	}

	logger.Debug().
		Interface("metric_last_state", metricLastState).
		Msg("Metric TTL expired for state")

	if triggerChecker.ttlState == moira.TTLStateDEL && metricLastState.EventTimestamp != 0 {
		if metricLastState.Maintenance != 0 && lastCheckTimeStamp <= metricLastState.Maintenance {
			metricLastState.DeletedButKept = true
			return false, &metricLastState
		}
		return true, nil
	}

	return false, newMetricState(
		metricLastState,
		triggerChecker.ttlState.ToMetricState(),
		lastCheckTimeStamp,
		map[string]float64{},
	)
}

func (triggerChecker *TriggerChecker) getMetricStepsStates(
	metricName string,
	metrics map[string]metricSource.MetricData,
	logger moira.Logger,
) (
	lastMetricState moira.MetricState,
	newMetricStates []moira.MetricState,
	err error,
) {
	var startTime int64
	var stepTime int64

	for _, metric := range metrics { // Taking values from any metric
		lastMetricState = triggerChecker.lastCheck.GetOrCreateMetricState(
			metricName,
			triggerChecker.trigger.MuteNewMetrics,
			checkPointGap,
		)

		startTime = metric.StartTime
		stepTime = metric.StepTime
		break
	}

	checkPoint := lastMetricState.GetCheckPoint(checkPointGap)
	logger.Debug().
		Int64(moira.LogFieldNameCheckpoint, checkPoint).
		Msg("Checkpoint got")

	// DO NOT CHANGE
	// Specific optimization magic
	previousMetricState := lastMetricState
	difference := moira.MaxInt64(checkPoint-startTime, 0)
	stepsDifference := difference / stepTime
	if (difference % stepTime) > 0 {
		stepsDifference++
	}
	valueTimestamp := startTime + stepTime*stepsDifference
	endTimestamp := triggerChecker.until + stepTime

	newMetricStates = make([]moira.MetricState, 0)

	for ; valueTimestamp < endTimestamp; valueTimestamp += stepTime {
		newMetricState, err := triggerChecker.getMetricDataState(metrics, &previousMetricState, &valueTimestamp, &checkPoint, logger)
		if err != nil {
			return lastMetricState, newMetricStates, err
		}

		if newMetricState == nil {
			continue
		}

		previousMetricState = *newMetricState
		newMetricStates = append(newMetricStates, *newMetricState)
	}

	return lastMetricState, newMetricStates, nil
}

func (triggerChecker *TriggerChecker) getMetricDataState(
	metrics map[string]metricSource.MetricData,
	lastState *moira.MetricState,
	valueTimestamp, checkPoint *int64,
	logger moira.Logger,
) (*moira.MetricState, error) {
	if *valueTimestamp <= *checkPoint {
		return nil, nil
	}

	triggerExpression, values, noEmptyValues := getExpressionValues(metrics, valueTimestamp, logger)
	if !noEmptyValues {
		return nil, nil
	}

	logger.Debug().
		Interface("timestamp", valueTimestamp).
		Interface("main_target_value", triggerExpression.MainTargetValue).
		Interface("additional_target_values", triggerExpression.AdditionalTargetsValues).
		Msg("Getting metric data state")

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
		*lastState,
		expressionState,
		*valueTimestamp,
		values,
	), nil
}

func getExpressionValues(
	metrics map[string]metricSource.MetricData,
	valueTimestamp *int64,
	logger moira.Logger,
) (
	triggerExpression *expression.TriggerExpression,
	values map[string]float64,
	noEmptyValues bool,
) {
	expression := &expression.TriggerExpression{
		AdditionalTargetsValues: make(map[string]float64, len(metrics)-1),
	}

	values = make(map[string]float64, len(metrics))

	for i := 0; i < len(metrics); i++ {
		targetName := fmt.Sprintf("t%d", i+1)
		metric, ok := metrics[targetName]
		if !ok {
			logger.Error().
				String("target", targetName).
				Msg("Failed to get metric by target")

			return nil, nil, false
		}

		value := metric.GetTimestampValue(*valueTimestamp)
		values[targetName] = value

		if !moira.IsFiniteNumber(value) {
			return nil, nil, false
		}

		if i == 0 {
			expression.MainTargetValue = value
			continue
		}

		expression.AdditionalTargetsValues[targetName] = value
	}

	return expression, values, true
}
