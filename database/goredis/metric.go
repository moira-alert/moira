package goredis

import (
	"encoding/json"
	"fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/goredis/reply"
	"github.com/patrickmn/go-cache"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// GetPatterns gets updated patterns array
func (connector *DbConnector) GetPatterns() ([]string, error) {
	c := *connector.client
	patterns, err := c.SMembers(connector.context, patternsListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get moira patterns, error: %v", err)
	}
	return patterns, nil
}

// GetMetricsValues gets metrics values for given interval
func (connector *DbConnector) GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*moira.MetricValue, error) {
	c := *connector.client
	resultByMetrics := make([]*redis.ZSliceCmd, 0, len(metrics))

	for _, metric := range metrics {
		rng := &redis.ZRangeBy{Min: strconv.FormatInt(from, 10), Max: strconv.FormatInt(until, 10)}
		result := c.ZRangeByScoreWithScores(connector.context, metricDataKey(metric), rng)
		resultByMetrics = append(resultByMetrics, result)
	}

	res := make(map[string][]*moira.MetricValue, len(resultByMetrics))

	for i, resultByMetric := range resultByMetrics {
		metric := metrics[i]
		metricsValues, err := reply.MetricValues(resultByMetric)
		if err != nil {
			return nil, err
		}
		res[metric] = metricsValues
	}
	return res, nil
}

// SaveMetrics saves new metrics
func (connector *DbConnector) SaveMetrics(metrics map[string]*moira.MatchedMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	c := *connector.client
	for _, metric := range metrics {
		metricValue := fmt.Sprintf("%v %v", metric.Timestamp, metric.Value)
		z := &redis.Z{Score: float64(metric.RetentionTimestamp), Member: metricValue}
		c.ZAdd(connector.context, metricDataKey(metric.Metric), z)

		if err := connector.retentionSavingCache.Add(metric.Metric, true, cache.DefaultExpiration); err == nil {
			c.Set(connector.context, metricRetentionKey(metric.Metric), metric.Retention, redis.KeepTTL)
		}

		for _, pattern := range metric.Patterns {
			c.SAdd(connector.context, patternMetricsKey(pattern), metric.Metric)
			event, err := json.Marshal(&moira.MetricEvent{
				Metric:  metric.Metric,
				Pattern: pattern,
			})
			if err != nil {
				continue
			}
			c.Publish(connector.context, metricEventKey, event)
		}
	}
	return nil
}

// GetPatternMetrics gets all metrics by given pattern
func (connector *DbConnector) GetPatternMetrics(pattern string) ([]string, error) {
	c := *connector.client

	metrics, err := c.SMembers(connector.context, patternMetricsKey(pattern)).Result()
	if err != nil {
		if err == redis.Nil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("failed to get pattern metrics for pattern %s, error: %v", pattern, err)
	}
	return metrics, nil
}

// RemovePatternWithMetrics removes pattern metrics with data and given pattern
func (connector *DbConnector) RemovePatternWithMetrics(pattern string) error {
	metrics, err := connector.GetPatternMetrics(pattern)
	if err != nil {
		return err
	}
	pipe := (*connector.client).TxPipeline()
	pipe.SRem(connector.context, patternsListKey, pattern)
	for _, metric := range metrics {
		pipe.Del(connector.context, metricDataKey(metric))
		pipe.Del(connector.context, metricRetentionKey(metric))
	}
	pipe.Del(connector.context, patternMetricsKey(pattern))
	if _, err = pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC: %v", err)
	}
	return nil
}

var patternsListKey = "moira-pattern-list"
var metricEventKey = "metric-event"

func patternMetricsKey(pattern string) string {
	return "moira-pattern-metrics:" + pattern
}

func metricDataKey(metric string) string {
	return "moira-metric-data:" + metric
}

func metricRetentionKey(metric string) string {
	return "moira-metric-retention:" + metric
}
