package redis

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
	"github.com/patrickmn/go-cache"
	"gopkg.in/tomb.v2"
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
	c := *connector.client

	retentionStr, err := c.Get(connector.context, metricRetentionKey(metric)).Result()
	if err != nil {
		if err == redis.Nil {
			return 60, database.ErrNil //nolint
		}
		return 0, fmt.Errorf("failed GET metric retention:%s, error: %v", metric, err)
	}
	retention, err := strconv.ParseInt(retentionStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed GET metric retention:%s, error: %v", metric, err)
	}
	return retention, nil
}

// SaveMetrics saves new metrics
func (connector *DbConnector) SaveMetrics(metrics map[string]*moira.MatchedMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	var err error
	c := *connector.client
	ctx := connector.context

	for _, metric := range metrics {
		metricValue := fmt.Sprintf("%v %v", metric.Timestamp, metric.Value)
		z := &redis.Z{Score: float64(metric.RetentionTimestamp), Member: metricValue}
		if err = c.ZAdd(ctx, metricDataKey(metric.Metric), z).Err(); err != nil {
			return err
		}

		if err = connector.retentionSavingCache.Add(metric.Metric, true, cache.DefaultExpiration); err == nil {
			if err = c.Set(ctx, metricRetentionKey(metric.Metric), metric.Retention, redis.KeepTTL).Err(); err != nil {
				return err
			}
		}

		for _, pattern := range metric.Patterns {
			if err = c.SAdd(ctx, patternMetricsKey(pattern), metric.Metric).Err(); err != nil {
				return err
			}

			var event []byte
			event, err = json.Marshal(&moira.MetricEvent{
				Metric:  metric.Metric,
				Pattern: pattern,
			})

			if err != nil {
				continue
			}

			rand.Seed(time.Now().UnixNano())
			var metricEventsChannel = metricEventsChannels[rand.Intn(len(metricEventsChannels))]
			if err = c.Publish(ctx, metricEventsChannel, event).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

// SubscribeMetricEvents creates subscription for new metrics and return channel for this events
func (connector *DbConnector) SubscribeMetricEvents(tomb *tomb.Tomb) (<-chan *moira.MetricEvent, error) {
	metricsChannel := make(chan *moira.MetricEvent, pubSubWorkerChannelSize)
	dataChannel, err := connector.manageSubscriptions(tomb, metricEventsChannels)
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
	c := *connector.client
	if _, err := c.SAdd(connector.context, patternMetricsKey(pattern), metric).Result(); err != nil {
		return fmt.Errorf("failed to SADD pattern-metrics, pattern: %s, metric: %s, error: %v", pattern, metric, err)
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

// RemovePattern removes pattern from patterns list
func (connector *DbConnector) RemovePattern(pattern string) error {
	c := *connector.client
	if _, err := c.SRem(connector.context, patternsListKey, pattern).Result(); err != nil {
		return fmt.Errorf("failed to remove pattern: %s, error: %v", pattern, err)
	}
	return nil
}

// RemovePatternsMetrics removes metrics by given patterns
func (connector *DbConnector) RemovePatternsMetrics(patterns []string) error {
	pipe := (*connector.client).TxPipeline()
	for _, pattern := range patterns {
		pipe.Del(connector.context, patternMetricsKey(pattern)) //nolint
	}
	if _, err := pipe.Exec(connector.context); err != nil {
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

// RemoveMetricValues remove metric timestamps values from 0 to given time
func (connector *DbConnector) RemoveMetricValues(metric string, toTime int64) error {
	if !connector.needRemoveMetrics(metric) {
		return nil
	}
	c := *connector.client
	if _, err := c.ZRemRangeByScore(connector.context, metricDataKey(metric), "-inf", strconv.FormatInt(toTime, 10)).Result(); err != nil {
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
	pipe := (*connector.client).TxPipeline()
	for _, metric := range metrics {
		if connector.needRemoveMetrics(metric) {
			pipe.ZRemRangeByScore(connector.context, metricDataKey(metric), "-inf", strconv.FormatInt(toTime, 10)) //nolint
		}
	}
	if _, err := pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC remove metrics: %v", err)
	}
	return nil
}

func (connector *DbConnector) needRemoveMetrics(metric string) bool {
	err := connector.metricsCache.Add(metric, true, 0)
	return err == nil
}

var patternsListKey = "moira-pattern-list"

var metricEventsChannels = []string{
	"metric-event-0",
	"metric-event-1",
	"metric-event-2",
	"metric-event-3",
}

func patternMetricsKey(pattern string) string {
	return "moira-pattern-metrics:" + pattern
}

func metricDataKey(metric string) string {
	return "moira-metric-data:" + metric
}

func metricRetentionKey(metric string) string {
	return "moira-metric-retention:" + metric
}
