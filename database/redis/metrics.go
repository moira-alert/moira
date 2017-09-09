package redis

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

// GetPatterns gets updated patterns array
func (connector *DbConnector) GetPatterns() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	patterns, err := redis.Strings(c.Do("SMEMBERS", patternsListKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to get moira patterns, error: %s", err)
	}
	return patterns, nil
}

// GetMetricsValues gets metrics values for given interval
func (connector *DbConnector) GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*moira.MetricValue, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, metric := range metrics {
		c.Send("ZRANGEBYSCORE", metricDataKey(metric), from, until, "WITHSCORES")
	}
	resultByMetrics, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	res := make(map[string][]*moira.MetricValue)

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
	retention, err := connector.readMetricRetention(metric)
	if err != nil {
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

func (connector *DbConnector) readMetricRetention(metric string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	retention, err := redis.Int64(c.Do("GET", metricRetentionKey(metric)))
	if err != nil {
		if err == redis.ErrNil {
			return 60, nil
		}
		return 0, fmt.Errorf("Failed GET metric-retention:%s, error: %s", metric, err.Error())
	}
	return retention, nil
}

// SaveMetrics saves new metrics
func (connector *DbConnector) SaveMetrics(metrics map[string]*moira.MatchedMetric) error {
	c := connector.pool.Get()
	defer c.Close()
	for _, metric := range metrics {

		metricValue := fmt.Sprintf("%v %v", metric.Timestamp, metric.Value)
		c.Send("ZADD", metricDataKey(metric.Metric), metric.RetentionTimestamp, metricValue)
		c.Send("SET", metricRetentionKey(metric.Metric), metric.Retention)

		for _, pattern := range metric.Patterns {
			event, err := json.Marshal(&moira.MetricEvent{
				Metric:  metric.Metric,
				Pattern: pattern,
			})
			if err != nil {
				continue
			}
			c.Send("PUBLISH", metricEventKey, event)
		}
	}
	return c.Flush()
}

// SubscribeMetricEvents creates subscription for new metrics and return channel for this events
func (connector *DbConnector) SubscribeMetricEvents(tomb *tomb.Tomb) <-chan *moira.MetricEvent {
	c := connector.pool.Get()
	psc := redis.PubSubConn{Conn: c}
	psc.Subscribe(metricEventKey)

	metricsChannel := make(chan *moira.MetricEvent, 100)
	dataChannel := connector.manageSubscriptions(psc)

	go func() {
		defer c.Close()
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from '%s' redis channel...", metricEventKey)
		psc.Unsubscribe()
	}()

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
				connector.logger.Errorf("Failed to parse MetricEvent: %s, error : %s", string(data), err.Error())
				continue
			}
			metricsChannel <- metricEvent
		}
	}()

	return metricsChannel
}

// AddPatternMetric adds new metrics by given pattern
func (connector *DbConnector) AddPatternMetric(pattern, metric string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SADD", patternMetricsKey(pattern), metric)
	if err != nil {
		return fmt.Errorf("Failed to SADD pattern-metrics, pattern: %s, metric: %s, error: %s", pattern, metric, err.Error())
	}
	return err
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
		return nil, fmt.Errorf("Failed to get pattern metrics for pattern %s, error: %s", pattern, err.Error())
	}
	return metrics, nil
}

// RemovePattern removes pattern from patterns list
func (connector *DbConnector) RemovePattern(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SREM", patternsListKey, pattern)
	if err != nil {
		return fmt.Errorf("Failed to remove pattern: %s, error: %s", pattern, err.Error())
	}
	return nil
}

// RemovePatternsMetrics removes metrics by given patterns
func (connector *DbConnector) RemovePatternsMetrics(patterns []string) error {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, pattern := range patterns {
		c.Send("DEL", patternMetricsKey(pattern))
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
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
	c.Send("MULTI")
	c.Send("SREM", patternsListKey, pattern)
	for _, metric := range metrics {
		c.Send("DEL", metricDataKey(metric))
	}
	c.Send("DEL", patternMetricsKey(pattern))
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

// RemoveMetricValues remove metrics timestamps values from 0 to given time
func (connector *DbConnector) RemoveMetricValues(metric string, toTime int64) error {
	if !connector.needRemoveMetrics(metric) {
		return nil
	}
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("ZREMRANGEBYSCORE", metricDataKey(metric), "-inf", toTime)
	if err != nil {
		return fmt.Errorf("Failed to remove metrics from -inf to %v, error: %s", toTime, err.Error())
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
	return fmt.Sprintf("moira-pattern-metrics:%s", pattern)
}

func metricDataKey(metric string) string {
	return fmt.Sprintf("moira-metric-data:%s", metric)
}

func metricRetentionKey(metric string) string {
	return fmt.Sprintf("moira-metric-retention:%s", metric)
}
