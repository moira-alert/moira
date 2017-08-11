package checker

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"math"
)

var checkPointGap int64 = 120

func (triggerChecker *TriggerChecker) Check() error {
	checkData, err := triggerChecker.handleTrigger()
	if err != nil {
		triggerChecker.Logger.Errorf("Trigger check failed: %s", err.Error())
		checkData = &moira.CheckData{
			Metrics:   triggerChecker.lastCheck.Metrics,
			State:     EXCEPTION,
			Timestamp: &triggerChecker.Until,
			Score:     triggerChecker.lastCheck.Score,
			Message:   "Trigger evaluation exception",
		}
		/*if err := triggerChecker.compareStates(metricState, *now, nil, nil); err != nil {
			return err
		}*/
		//todo compare_states
		return nil //todo is it right?
	}

	checkData.Score = scores[checkData.State]
	for _, metricData := range checkData.Metrics {
		checkData.Score += scores[metricData.State]
	}
	triggerChecker.Database.SetTriggerLastCheck(triggerChecker.TriggerId, checkData)
	return nil
}

func (triggerChecker *TriggerChecker) handleTrigger() (*moira.CheckData, error) {
	checkData := &moira.CheckData{
		Metrics:   triggerChecker.lastCheck.Metrics,
		State:     OK,
		Timestamp: &triggerChecker.Until,
		Score:     triggerChecker.lastCheck.Score,
	}

	triggerTimeSeries, metrics, err := triggerChecker.getTimeSeries(triggerChecker.From, triggerChecker.Until)
	if err != nil {
		return checkData, err
	}

	triggerChecker.cleanupMetricsValues(metrics, triggerChecker.Until)

	if len(triggerTimeSeries.Main) == 0 {
		if triggerChecker.ttl != nil && len(triggerChecker.lastCheck.Metrics) != 0 {
			checkData.State = triggerChecker.ttlState
			checkData.Message = "Trigger has no metrics"
			/*if err := triggerChecker.compareStates(metricState, until, nil, nil); err != nil {
				return checkData, err
			}*/
		}
		return checkData, nil
	}

	for _, timeSeries := range triggerTimeSeries.Main {
		startTime := int64(timeSeries.StartTime)
		stopTime := int64(timeSeries.StopTime)
		stepTime := int64(timeSeries.StepTime)

		triggerChecker.Logger.Debugf("Checking timeSeries %s: %v", timeSeries.Name, timeSeries.Values)
		triggerChecker.Logger.Debugf("Checking interval: %v - %v (%vs), step: %v", startTime, stopTime, stepTime, stopTime-startTime)
		metricState, ok := checkData.Metrics[timeSeries.Name]
		if !ok {
			metricState = moira.MetricState{
				State:     NODATA,
				Timestamp: startTime - 3600,
			}
		}

		checkPoint := getCheckPoint(metricState, checkPointGap)
		triggerChecker.Logger.Debugf("Checkpoint for %s: %v", timeSeries.Name, checkPoint)

		for valueTimestamp := startTime; valueTimestamp < triggerChecker.Until+stepTime; valueTimestamp += stepTime {
			if valueTimestamp < checkPoint {
				continue
			}
			expressionValues, noEmptyValues := triggerTimeSeries.getExpressionValues(timeSeries, checkPoint)
			triggerChecker.Logger.Debugf("values for ts %s: %v", valueTimestamp, expressionValues)
			if noEmptyValues {
				continue
			}

			expressionValues["warn_value"] = getMathFloat64Pointer(triggerChecker.trigger.WarnValue)
			expressionValues["error_value"] = getMathFloat64Pointer(triggerChecker.trigger.ErrorValue)
			expressionValues["PREV_STATE"] = 1000 //todo NODATA

			expressionState := GetExpression(triggerChecker.trigger.Expression, expressionValues)
			triggerTimeSeries.updateCheckData(timeSeries, checkData, expressionState, expressionValues, valueTimestamp)
			t1Value := expressionValues[triggerTimeSeries.getMainTargetName()]
			if err := triggerChecker.compareStates(metricState, valueTimestamp, &t1Value, &timeSeries.Name); err != nil {
				return checkData, err
			}
			//todo compare_states
		}

		lastCheckTimeStamp := triggerChecker.lastCheck.Timestamp
		ttl := triggerChecker.ttl

		//compare with last_check timestamp in case if we have not run checker for a long time
		if ttl != nil && metricState.Timestamp+*triggerChecker.ttl < moira.UseInt64(lastCheckTimeStamp) {
			triggerChecker.Logger.Infof("Metric %s TTL expired for state %v", timeSeries.Name, metricState)
			if triggerChecker.ttlState == DEL && metricState.EventTimestamp != 0 {
				triggerChecker.Logger.Infof("Remove metric %s", timeSeries.Name)
				delete(checkData.Metrics, timeSeries.Name)
				if err := triggerChecker.Database.RemovePatternsMetrics(triggerChecker.trigger.Patterns); err != nil {
					return checkData, err
				}
				continue
			}
			triggerTimeSeries.updateCheckData(timeSeries, checkData, toMetricState(triggerChecker.ttlState), nil, *lastCheckTimeStamp-*ttl)
			if err := triggerChecker.compareStates(metricState, metricState.Timestamp, nil, &timeSeries.Name); err != nil {
				return checkData, err
			}
		}
	}
	return checkData, nil
}

func getCheckPoint(metricData moira.MetricState, checkPointGap int64) int64 {
	return int64(math.Max(float64(metricData.Timestamp-checkPointGap), float64(metricData.EventTimestamp)))
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

func (triggerChecker *TriggerChecker) getTimeSeries(from, until int64) (*triggerTimeSeries, []string, error) {
	targets := triggerChecker.trigger.Targets
	triggerTimeSeries := &triggerTimeSeries{
		Main:       make([]*TimeSeries, 0),
		Additional: make([]*TimeSeries, 0),
	}
	metricsArr := make([]string, 0)

	for targetIndex, target := range targets {
		metricDatas, metrics, err := EvaluateTarget(triggerChecker.Database, target, from, until, triggerChecker.isSimple)
		if err != nil {
			return nil, nil, err
		}

		if targetIndex == 0 {
			triggerTimeSeries.Main = metricDatas
		} else {
			if len(metricDatas) == 0 {
				return nil, nil, fmt.Errorf("Target #%v has no timeseries", targetIndex+1)
			} else if len(metricDatas) > 1 {
				return nil, nil, fmt.Errorf("Target #%v has more than one timeseries", targetIndex+1)
			}
			triggerTimeSeries.Additional = append(triggerTimeSeries.Additional, metricDatas[0])
		}
		metricsArr = append(metricsArr, metrics...)
	}
	return triggerTimeSeries, metricsArr, nil
}

func getMathFloat64Pointer(val *float64) float64 {
	if val != nil {
		return *val
	} else {
		return math.NaN()
	}
}
