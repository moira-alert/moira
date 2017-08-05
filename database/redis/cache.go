package redis

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
)

func makeEvent(pattern string, metric string) ([]byte, error) {
	return json.Marshal(&moira.MetricEvent{
		Metric:  metric,
		Pattern: pattern,
	})
}

// UpdateMetricsHeartbeat increments redis counter
func (connector *DbConnector) UpdateMetricsHeartbeat() error {
	c := connector.pool.Get()
	defer c.Close()
	err := c.Send("INCR", "moira-selfstate:metrics-heartbeat")
	return err
}

// GetPatterns gets updated patterns array
func (connector *DbConnector) GetPatterns() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	return redis.Strings(c.Do("SMEMBERS", "moira-pattern-list"))
}

//SaveMetrics saves new metrics
func (connector *DbConnector) SaveMetrics(buffer map[string]*moira.MatchedMetric) error {

	c := connector.pool.Get()
	defer c.Close()

	for _, m := range buffer {

		metricKey := getMetricDbKey(m.Metric)
		metricRetentionKey := getMetricRetentionDbKey(m.Metric)

		metricValue := fmt.Sprintf("%v %v", m.Timestamp, m.Value)

		c.Send("ZADD", metricKey, m.RetentionTimestamp, metricValue)
		c.Send("SET", metricRetentionKey, m.Retention)

		for _, pattern := range m.Patterns {
			event, err := makeEvent(pattern, m.Metric)
			if err != nil {
				continue
			}
			c.Send("PUBLISH", "metric-event", event)
		}
	}
	return c.Flush()
}

// getMetricDbKey returns string redis key for metric
func getMetricDbKey(metric string) string {
	return fmt.Sprintf("moira-metric-data:%s", metric)
}

// getMetricRetentionDbKey returns string redis key for metric retention
func getMetricRetentionDbKey(metric string) string {
	return fmt.Sprintf("moira-metric-retention:%s", metric)
}
