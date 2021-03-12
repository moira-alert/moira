package support

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/metric_source/local"
)

const defaultRetention = 10

func HandlePullTrigger(logger moira.Logger, database moira.Database, triggerID string) (*moira.Trigger, error) {
	logger.Infof("Pull database info about trigger %s", triggerID)

	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return nil, fmt.Errorf("cannot get trigger: %w", err)
	}
	return &trigger, nil
}

func HandlePullTriggerMetrics(logger moira.Logger, database moira.Database, triggerID string) ([]dto.PatternMetrics, error) {
	logger.Infof("Pulling info about trigger %s metrics", triggerID)
	source := local.Create(database)

	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return nil, fmt.Errorf("cannot get trigger: %w", err)
	}
	ttl := database.GetMetricsTTLSeconds()
	until := time.Now().Unix()
	from := until - ttl
	result := []dto.PatternMetrics{}
	for _, target := range trigger.Targets {
		fetchResult, errFetch := source.Fetch(target, from, until, trigger.IsSimple())
		if errFetch != nil {
			return nil, fmt.Errorf("cannot fetch metrics for target %s: %w", target, errFetch)
		}
		patterns, errPatterns := fetchResult.GetPatterns()
		if errPatterns != nil {
			return nil, fmt.Errorf("cannot get patterns for target %s: %w", target, errPatterns)
		}
		for _, pattern := range patterns {
			patternResult := dto.PatternMetrics{
				Pattern:    pattern,
				Retentions: make(map[string]int64),
			}
			metrics, errMetrics := database.GetPatternMetrics(pattern)
			if errMetrics != nil {
				return nil, fmt.Errorf("cannot get metrics for pattern %s, target %s: %w", pattern, target, errMetrics)
			}
			for _, metric := range metrics {
				retention, errRetention := database.GetMetricRetention(metric)
				if errRetention != nil {
					return nil, fmt.Errorf("cannot get metric %s retention: %w", metric, errRetention)
				}
				patternResult.Retentions[metric] = retention
			}
			values, errValues := database.GetMetricsValues(metrics, from, until)
			if errValues != nil {
				return nil, fmt.Errorf("cannot get values for pattern %s metrics, target %s: %w", pattern, target, errValues)
			}
			patternResult.Metrics = values
			result = append(result, patternResult)
		}
	}
	return result, nil
}

func HandlePushTrigger(logger moira.Logger, database moira.Database, trigger *moira.Trigger) error {
	logger.Info("Save trigger")
	err := database.SaveTrigger(trigger.ID, trigger)
	if err != nil {
		return fmt.Errorf("cannot save trigger: %w", err)
	}
	logger.Infof("Trigger %s was saved", trigger.ID)
	return nil
}

func HandlePushTriggerMetrics(logger moira.Logger, database moira.Database, triggerID string, patternsMetrics []dto.PatternMetrics) error {
	logger.Info("Save trigger metrics")

	buffer := make(map[string]*moira.MatchedMetric, len(patternsMetrics))
	i := 0
	for _, patternMetrics := range patternsMetrics {
		for metricName, metricValues := range patternMetrics.Metrics {
			for _, metricValue := range metricValues {
				i++
				retention, ok := patternMetrics.Retentions[metricName]
				if !ok {
					retention = defaultRetention
				}
				matchedMetric := moira.MatchedMetric{
					Patterns: []string{
						patternMetrics.Pattern,
					},
					Metric:             metricName,
					Value:              metricValue.Value,
					Timestamp:          metricValue.Timestamp,
					RetentionTimestamp: metricValue.RetentionTimestamp,
					Retention:          int(retention),
				}
				buffer[fmt.Sprintf("%d", i)] = &matchedMetric
			}
		}
	}
	err := database.SaveMetrics(buffer)
	if err != nil {
		return fmt.Errorf("cannot save trigger metrics: %w", err)
	}
	logger.Infof("Trigger %s metrics was saved", triggerID)
	return nil
}

func HandlePushTriggerLastCheck(logger moira.Logger, database moira.Database, triggerID string,
	lastCheck *moira.CheckData, isRemoteTrigger bool) error {
	logger.Info("Save trigger last check")
	if err := database.SetTriggerLastCheck(triggerID, lastCheck, isRemoteTrigger); err != nil {
		return fmt.Errorf("cannot set trigger last check: %w", err)
	}
	logger.Info("Trigger last check was saved")
	return nil
}
