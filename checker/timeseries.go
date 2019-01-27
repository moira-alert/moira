package checker

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/expression"
	"github.com/moira-alert/moira/metric_source"
)

func (triggerChecker *TriggerChecker) getFetchResult() (*metricSource.TriggerMetricsData, []string, error) {
	wrongTriggerTargets := make([]int, 0)
	triggerMetricsData := metricSource.MakeEmptyTriggerMetricsData()
	metricsArr := make([]string, 0)

	isSimpleTrigger := triggerChecker.trigger.IsSimple()
	for targetIndex, target := range triggerChecker.trigger.Targets {
		fetchResult, err := triggerChecker.source.Fetch(target, triggerChecker.from, triggerChecker.until, isSimpleTrigger)
		if err != nil {
			return nil, nil, err
		}
		metricsData := fetchResult.GetMetricsData()
		metricsFetchResult, metricsErr := fetchResult.GetPatternMetrics()
		if targetIndex == 0 {
			triggerMetricsData.Main = metricsData
		} else {
			metricsDataCount := len(metricsData)
			switch {
			case metricsDataCount == 0:
				if metricsErr != nil {
					return nil, nil, ErrTargetHasNoTimeSeries{targetIndex: targetIndex + 1}
				}
				if len(metricsFetchResult) == 0 {
					triggerMetricsData.Additional = append(triggerMetricsData.Additional, nil)
				} else {
					return nil, nil, ErrTargetHasNoTimeSeries{targetIndex: targetIndex + 1}
				}
			case metricsDataCount > 1:
				wrongTriggerTargets = append(wrongTriggerTargets, targetIndex+1)
			default:
				triggerMetricsData.Additional = append(triggerMetricsData.Additional, metricsData[0])
			}
		}
		if metricsErr == nil {
			metricsArr = append(metricsArr, metricsFetchResult...)
		}
	}
	if len(wrongTriggerTargets) > 0 {
		return nil, nil, ErrWrongTriggerTargets(wrongTriggerTargets)
	}
	return triggerMetricsData, metricsArr, nil
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
		additionalTimeSeries := triggerMetricsData.Additional[targetNumber]
		if additionalTimeSeries == nil {
			return expressionValues, false
		}
		tnValue := additionalTimeSeries.GetTimestampValue(valueTimestamp)
		if !moira.IsValidFloat64(tnValue) {
			return expressionValues, false
		}
		expressionValues.AdditionalTargetsValues[triggerMetricsData.GetAdditionalTargetName(targetNumber)] = tnValue
	}
	return expressionValues, true
}
