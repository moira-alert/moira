package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

type patternMetrics struct {
	Pattern    string                          `json:"pattern"`
	Metrics    map[string][]*moira.MetricValue `json:"metrics"`
	Retentions map[string]int64                `json:"retention"`
}

const defaultRetention = 10

func handlePullTrigger(logger moira.Logger, database moira.Database, triggerID string, filePath string) error {
	logger.Infof("Save info about trigger %s", triggerID)

	if filePath == "" {
		return fmt.Errorf("file is not specified")
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)

	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return fmt.Errorf("cannot get trigger: %w", err)
	}
	err = encoder.Encode(trigger)
	if err != nil {
		return fmt.Errorf("cannot marshall trigger: %w", err)
	}
	return nil
}

func handlePullTriggerMetrics(source metricSource.MetricSource, logger moira.Logger, database moira.Database, triggerID string, filePath string) error {
	logger.Infof("Pulling info about trigger %s metrics", triggerID)

	if filePath == "" {
		return fmt.Errorf("file is not specified")
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)

	trigger, err := database.GetTrigger(triggerID)
	if err != nil {
		return fmt.Errorf("cannot get trigger: %w", err)
	}
	ttl := database.GetMetricsTTLSeconds()
	until := time.Now().Unix()
	from := until - ttl
	result := []patternMetrics{}
	for _, target := range trigger.Targets {
		fetchResult, err := source.Fetch(target, from, until, trigger.IsSimple())
		if err != nil {
			return fmt.Errorf("cannot fetch metrics for target %s: %w", target, err)
		}
		patterns, err := fetchResult.GetPatterns()
		if err != nil {
			return fmt.Errorf("cannot get patterns for target %s: %w", target, err)
		}
		for _, pattern := range patterns {
			patternResult := patternMetrics{
				Pattern:    pattern,
				Retentions: make(map[string]int64),
			}
			metrics, err := database.GetPatternMetrics(pattern)
			if err != nil {
				return fmt.Errorf("cannot get metrics for pattern %s, target %s: %w", pattern, target, err)
			}
			for _, metric := range metrics {
				retention, err := database.GetMetricRetention(metric)
				if err != nil {
					return fmt.Errorf("cannot get metric %s retention: %w", metric, err)
				}
				patternResult.Retentions[metric] = retention
			}
			values, err := database.GetMetricsValues(metrics, from, until)
			if err != nil {
				return fmt.Errorf("cannot get values for pattern %s metrics, target %s: %w", pattern, target, err)
			}
			patternResult.Metrics = values
			result = append(result, patternResult)
		}
	}
	err = encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("cannot marshall trigger metrics: %w", err)
	}
	logger.Info("Metrics pulled")
	return nil
}

func handlePushTrigger(logger moira.Logger, database moira.Database, filePath string) error {
	logger.Info("Reading trigger JSON from file")
	if filePath == "" {
		return fmt.Errorf("file is not specified")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	trigger := &moira.Trigger{}
	err = decoder.Decode(trigger)
	if err != nil {
		return fmt.Errorf("cannot decode trigger: %w", err)
	}
	err = database.SaveTrigger(trigger.ID, trigger)
	if err != nil {
		return fmt.Errorf("cannot save trigger: %w", err)
	}
	logger.Infof("Trigger %s was saved", trigger.ID)
	return nil
}

func handlePushTriggerMetrics(logger moira.Logger, database moira.Database, triggerID string, filePath string) error {
	logger.Infof("Reading trigger metrics JSON from stdin")

	if filePath == "" {
		return fmt.Errorf("file is not specified")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	_, err = database.GetTrigger(triggerID)
	if err != nil {
		return fmt.Errorf("cannot get trigger: %w", err)
	}
	patternsMetrics := []patternMetrics{}
	err = decoder.Decode(&patternsMetrics)
	if err != nil {
		return fmt.Errorf("cannot decode trigger: %w", err)
	}
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
	err = database.SaveMetrics(buffer)
	if err != nil {
		return fmt.Errorf("cannot save trigger metrics: %w", err)
	}
	logger.Infof("Trigger %s metrics was saved", triggerID)
	return nil
}
