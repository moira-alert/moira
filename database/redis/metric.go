package redis

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
	"github.com/patrickmn/go-cache"
)

// GetPatterns gets updated patterns array
func (connector *DbConnector) GetPatterns() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	patterns, err := redis.Strings(c.Do("SMEMBERS", patternsListKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get moira patterns, error: %v", err)
	}
	return patterns, nil
}

// GetMetricsValues gets metrics values for given interval
func (connector *DbConnector) GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*moira.MetricValue, error) {
	c := connector.pool.Get()
	defer c.Close()

	for _, metric := range metrics {
		c.Send("ZRANGEBYSCORE", metricDataKey(metric), from, until, "WITHSCORES") //nolint
	}
	resultByMetrics, err := redis.Values(c.Do(""))
	if err != nil {
		return nil, database.ErrDatabase{Err: fmt.Errorf("failed to get metric values: %v", err)}
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

// GetMetricRetention gets given metric retention, if retention is empty then return default retention value(60)
func (connector *DbConnector) GetMetricRetention(metric string) (int64, error) {
	retention, ok := connector.getCachedRetention(metric)
	if ok {
		return retention, nil
	}
	retention, err := connector.getMetricRetention(metric)
	if err != nil {
		if err == database.ErrNil {
			return retention, nil
		}
		return retention, err
	}
	connector.retentionCache.Set(metric, retention, 0)
	return retention, nil
}

func (connector *DbConnector) getCachedRetention(metric string) (int64, bool) {
	value, ok := connector.retentionCache.Get(metric)
	if !ok {
		return 0, false
	}
	retention, ok := value.(int64)
	return retention, ok
}

func (connector *DbConnector) getMetricRetention(metric string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	retention, err := redis.Int64(c.Do("GET", metricRetentionKey(metric)))

	if err != nil {
		if err == redis.ErrNil {
			return 60, database.ErrNil //nolint
		}
		return 0, database.ErrDatabase{
			Err: fmt.Errorf("failed GET metric retention: %s, redis error: %v", metric, err),
		}
	}
	return retention, nil
}

// SaveMetrics saves new metrics
func (connector *DbConnector) SaveMetrics(metrics map[string]*moira.MatchedMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	c := connector.pool.Get()
	defer c.Close()
	for _, metric := range metrics {
		metricValue := fmt.Sprintf("%v %v", metric.Timestamp, metric.Value)
		c.Send("ZADD", metricDataKey(metric.Metric), metric.RetentionTimestamp, metricValue) //nolint

		if err := connector.retentionSavingCache.Add(metric.Metric, true, cache.DefaultExpiration); err == nil {
			c.Send("SET", metricRetentionKey(metric.Metric), metric.Retention) //nolint
		}

		for _, pattern := range metric.Patterns {
			c.Send("SADD", patternMetricsKey(pattern), metric.Metric) //nolint
			event, err := json.Marshal(&moira.MetricEvent{
				Metric:  metric.Metric,
				Pattern: pattern,
			})
			if err != nil {
				continue
			}
			c.Send("PUBLISH", metricEventKey, event) //nolint
		}
	}
	return c.Flush()
}

// SubscribeMetricEvents creates subscription for new metrics and return channel for this events
func (connector *DbConnector) SubscribeMetricEvents(tomb *tomb.Tomb) (<-chan *moira.MetricEvent, error) {
	metricsChannel := make(chan *moira.MetricEvent, pubSubWorkerChannelSize)
	dataChannel, err := connector.manageSubscriptions(tomb, metricEventKey)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			data, ok := <-dataChannel
			if !ok {
				connector.logger.Info("No more subscriptions, channel is closed. Stop process data...")
				close(metricsChannel)
				return
			}
			metricEvent := &moira.MetricEvent{}
			if err := json.Unmarshal(data, metricEvent); err != nil {
				connector.logger.Errorf("Failed to parse MetricEvent: %s, error : %v", string(data), err)
				continue
			}
			metricsChannel <- metricEvent
		}
	}()

	return metricsChannel, nil
}

// AddPatternMetric adds new metrics by given pattern
func (connector *DbConnector) AddPatternMetric(pattern, metric string) error {
	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("SADD", patternMetricsKey(pattern), metric); err != nil {
		return fmt.Errorf("failed to SADD pattern-metrics, pattern: %s, metric: %s, error: %v", pattern, metric, err)
	}
	return nil
}

// GetPatternMetrics gets all metrics by given pattern
func (connector *DbConnector) GetPatternMetrics(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	metrics, err := redis.Strings(c.Do("SMEMBERS", patternMetricsKey(pattern)))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), nil
		}
		return nil, database.ErrDatabase{
			Err: fmt.Errorf("failed to get pattern metrics for pattern %s, error: %v", pattern, err),
		}
	}
	return metrics, nil
}

// RemovePattern removes pattern from patterns list
func (connector *DbConnector) RemovePattern(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("SREM", patternsListKey, pattern); err != nil {
		return fmt.Errorf("failed to remove pattern: %s, error: %v", pattern, err)
	}
	return nil
}

// RemovePatternsMetrics removes metrics by given patterns
func (connector *DbConnector) RemovePatternsMetrics(patterns []string) error {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI") //nolint
	for _, pattern := range patterns {
		c.Send("DEL", patternMetricsKey(pattern)) //nolint
	}
	if _, err := c.Do("EXEC"); err != nil {
		return fmt.Errorf("failed to EXEC: %v", err)
	}
	return nil
}

// RemovePatternWithMetrics removes pattern metrics with data and given pattern
func (connector *DbConnector) RemovePatternWithMetrics(pattern string) error {
	metrics, err := connector.GetPatternMetrics(pattern)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")                          //nolint
	c.Send("SREM", patternsListKey, pattern) //nolint
	for _, metric := range metrics {
		c.Send("DEL", metricDataKey(metric))      //nolint
		c.Send("DEL", metricRetentionKey(metric)) //nolint
	}
	c.Send("DEL", patternMetricsKey(pattern)) //nolint
	if _, err = c.Do("EXEC"); err != nil {
		return fmt.Errorf("failed to EXEC: %v", err)
	}
	return nil
}

// RemoveMetricValues remove metric timestamps values from 0 to given time
func (connector *DbConnector) RemoveMetricValues(metric string, toTime int64) error {
	if !connector.needRemoveMetrics(metric) {
		return nil
	}
	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("ZREMRANGEBYSCORE", metricDataKey(metric), "-inf", toTime); err != nil {
		return fmt.Errorf("failed to remove metrics from -inf to %v, error: %v", toTime, err)
	}
	return nil
}

// GetMetricsTTLSeconds returns maximum time in seconds to store metrics in Redis
func (connector *DbConnector) GetMetricsTTLSeconds() int64 {
	return connector.metricsTTLSeconds
}

// RemoveMetricsValues remove metrics timestamps values from 0 to given time
func (connector *DbConnector) RemoveMetricsValues(metrics []string, toTime int64) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI") //nolint
	for _, metric := range metrics {
		if connector.needRemoveMetrics(metric) {
			c.Send("ZREMRANGEBYSCORE", metricDataKey(metric), "-inf", toTime) //nolint
		}
	}
	if _, err := c.Do("EXEC"); err != nil {
		return fmt.Errorf("failed to EXEC remove metrics: %v", err)
	}
	return nil
}

func (connector *DbConnector) needRemoveMetrics(metric string) bool {
	err := connector.metricsCache.Add(metric, true, 0)
	return err == nil
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
