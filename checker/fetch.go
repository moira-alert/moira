package checker

import (
	metricSource "github.com/moira-alert/moira/metric_source"
)

func (triggerChecker *TriggerChecker) fetchTriggerMetrics() (*metricSource.TriggerMetricsData, error) {
	triggerMetricsData, metrics, err := triggerChecker.fetch()
	if err != nil {
		return triggerMetricsData, err
	}
	triggerChecker.cleanupMetricsValues(metrics, triggerChecker.until)

	if len(triggerChecker.lastCheck.Metrics) == 0 {
		if len(triggerMetricsData.Main) == 0 {
			return triggerMetricsData, ErrTriggerHasNoMetrics{}
		}

		if triggerMetricsData.HasOnlyWildcards() {
			return triggerMetricsData, ErrTriggerHasOnlyWildcards{}
		}
	}

	return triggerMetricsData, nil
}

func (triggerChecker *TriggerChecker) fetch() (*metricSource.TriggerMetricsData, []string, error) {
	wrongTriggerTargets := make([]int, 0)
	triggerMetricsData := metricSource.NewTriggerMetricsData()
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
					return nil, nil, ErrTargetHasNoMetrics{targetIndex: targetIndex + 1}
				}
				if len(metricsFetchResult) == 0 {
					triggerMetricsData.Additional = append(triggerMetricsData.Additional, nil)
				} else {
					return nil, nil, ErrTargetHasNoMetrics{targetIndex: targetIndex + 1}
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

func (triggerChecker *TriggerChecker) cleanupMetricsValues(metrics []string, until int64) {
	if len(metrics) > 0 {
		if err := triggerChecker.database.RemoveMetricsValues(metrics, until-triggerChecker.database.GetMetricsTTLSeconds()); err != nil {
			triggerChecker.logger.Error(err.Error())
		}
	}
}
