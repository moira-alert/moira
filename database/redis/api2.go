package redis

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"gopkg.in/tomb.v2"
	"strconv"
	"strings"
	"time"
)

func (connector *DbConnector) GetPatternTriggerIds(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIds, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("moira-pattern-triggers:%s", pattern)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve pattern-triggers for pattern %s: %s", pattern, err.Error())
	}
	return triggerIds, nil
}

func (connector *DbConnector) GetTriggers(triggerIds []string) ([]*moira.Trigger, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerId := range triggerIds {
		c.Send("GET", fmt.Sprintf("moira-trigger:%s", triggerId))
		c.Send("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerId))
	}
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	triggers := make([]*moira.Trigger, 0)
	for i := 0; i < len(rawResponse); i += 2 {
		triggerSE, err := connector.convertTriggerWithTags(rawResponse[i], rawResponse[i+1], triggerIds[i/2])
		if err != nil {
			return nil, err
		}
		if triggerSE == nil {
			continue
		}
		triggers = append(triggers, toTrigger(triggerSE, triggerIds[i/2]))
	}

	return triggers, nil
}

func (connector *DbConnector) GetPatternMetrics(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	metrics, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("moira-pattern-metrics:%s", pattern)))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("Failed to retrieve pattern-metrics for pattern %s: %s", pattern, err.Error())
	}
	return metrics, nil
}

func (connector *DbConnector) RemovePattern(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SREM", "moira-pattern-list", pattern)
	if err != nil {
		return fmt.Errorf("Failed to remove pattern: %s, error: %s", pattern, err.Error())
	}
	return nil
}

func (connector *DbConnector) GetTriggerIds() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIds, err := redis.Strings(c.Do("SMEMBERS", "moira-triggers-list"))
	if err != nil {
		return nil, fmt.Errorf("Failed to get triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

func (connector *DbConnector) DeleteTriggerThrottling(triggerId string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SET", fmt.Sprintf("moira-notifier-throttling-beginning:%s", triggerId), time.Now().Unix())
	c.Send("DEL", fmt.Sprintf("moira-notifier-next:%s", triggerId))
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) DeleteTrigger(triggerId string) error {
	trigger, err := connector.GetTrigger(triggerId)
	if err != nil {
		return nil
	}
	if trigger == nil {
		return nil
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", fmt.Sprintf("moira-trigger:%s", triggerId))
	c.Send("DEL", fmt.Sprintf("moira-trigger-tags:%s", triggerId))
	c.Send("SREM", "moira-triggers-list", triggerId)
	for _, tag := range trigger.Tags {
		c.Send("SREM", fmt.Sprintf("moira-tag-triggers:%s", tag), triggerId)
	}
	for _, pattern := range trigger.Patterns {
		c.Send("SREM", fmt.Sprintf("moira-pattern-triggers:%s", pattern), triggerId)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	for _, pattern := range trigger.Patterns {
		count, err := redis.Int64(c.Do("SCARD", fmt.Sprintf("moira-pattern-triggers:%s", pattern)))
		if err != nil {
			return fmt.Errorf("Failed to SCARD pattern-triggers: %s", err.Error())
		}
		if count == 0 {
			if err := connector.RemovePatternWithMetrics(pattern); err != nil {
				return err
			}
		}
	}
	return nil
}

func (connector *DbConnector) RemovePatternWithMetrics(pattern string) error {
	metrics, err := connector.GetPatternMetrics(pattern)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SREM", "moira-pattern-list", pattern)
	for _, metric := range metrics {
		c.Send("DEL", fmt.Sprintf("moira-metric-data:%s", metric))
	}
	c.Send("DEL", fmt.Sprintf("moira-pattern-metrics:%s", pattern))
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) SetTriggerCheckLock(triggerId string) (bool, error) {
	c := connector.pool.Get()
	defer c.Close()
	_, err := redis.String(c.Do("SET", fmt.Sprintf("moira-metric-check-lock:%s", triggerId), time.Now().Unix(), "EX", 30, "NX"))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		return false, fmt.Errorf("Failed to set metric-check-lock:%s : %s", triggerId, err.Error())
	}
	return true, nil
}

func (connector *DbConnector) DeleteTriggerCheckLock(triggerId string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", fmt.Sprintf("moira-metric-check-lock:%s", triggerId))
	if err != nil {
		return fmt.Errorf("Failed to delete metric-check-lock:%s : %s", triggerId, err.Error())
	}
	return nil
}

func (connector *DbConnector) AcquireTriggerCheckLock(triggerId string, timeout int) error {
	acquired, err := connector.SetTriggerCheckLock(triggerId)
	if err != nil {
		return err
	}
	count := 0
	for !acquired && count < timeout {
		count += 1
		<-time.After(time.Millisecond * 500)
		acquired, err = connector.SetTriggerCheckLock(triggerId)
		if err != nil {
			return err
		}
	}
	if !acquired {
		return fmt.Errorf("Can not acquire trigger lock in %v seconds", timeout)
	}
	return nil
}

func (connector *DbConnector) SetTriggerLastCheck(triggerId string, checkData *moira.CheckData) error {
	bytes, err := json.Marshal(checkData)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SET", fmt.Sprintf("moira-metric-last-check:%s", triggerId), bytes)
	c.Send("ZADD", "moira-triggers-checks", checkData.Score, triggerId)
	c.Send("INCR", "moira-selfstate:checks-counter")
	if checkData.Score > 0 {
		c.Send("SADD", "moira-bad-state-triggers", triggerId)
	} else {
		c.Send("SREM", "moira-bad-state-triggers", triggerId)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) RemovePatternsMetrics(patterns []string) error {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, pattern := range patterns {
		c.Send("DEL", fmt.Sprintf("moira-pattern-metrics:%s", pattern))
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) SaveTrigger(triggerId string, trigger *moira.Trigger) error {
	existing, err := connector.GetTrigger(triggerId)
	if err != nil {
		return err
	}

	triggerSE := toTriggerStorageElement(trigger, triggerId)
	bytes, err := json.Marshal(triggerSE)
	if err != nil {
		return nil
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	cleanupPatterns := make([]string, 0)
	if existing != nil {
		for _, pattern := range leftJoin(existing.Patterns, trigger.Patterns) {
			c.Send("SREM", fmt.Sprintf("moira-pattern-triggers:%s", pattern), triggerId)
			cleanupPatterns = append(cleanupPatterns, pattern)
		}
		for _, tag := range leftJoin(existing.Tags, trigger.Tags) {
			c.Send("SREM", fmt.Sprintf("moira-trigger-tags:%s", triggerId), tag)
			c.Send("SREM", fmt.Sprintf("moira-tag-triggers:%s", tag), triggerId)
		}
	}
	c.Do("SET", fmt.Sprintf("moira-trigger:%s", triggerId), bytes)
	c.Do("SADD", "moira-triggers-list", triggerId)
	for _, pattern := range trigger.Patterns {
		c.Do("SADD", "moira-pattern-list", pattern)
		c.Do("SADD", fmt.Sprintf("moira-pattern-triggers:%s", pattern), triggerId)
	}
	for _, tag := range trigger.Tags {
		c.Send("SADD", fmt.Sprintf("moira-trigger-tags:%s", triggerId), tag)
		c.Send("SADD", fmt.Sprintf("moira-tag-triggers:%s", tag), triggerId)
		c.Send("SADD", "moira-tags", tag)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	for _, pattern := range cleanupPatterns {
		connector.RemovePatternTriggers(pattern)
		connector.RemovePattern(pattern)
		connector.RemovePatternsMetrics([]string{pattern})
	}
	return nil
}

func (connector *DbConnector) RemovePatternTriggers(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", fmt.Sprintf("moira-pattern-triggers:%s", pattern))
	if err != nil {
		return fmt.Errorf("Failed delete pattern-triggers: %s, error: %s", pattern, err)
	}
	return nil
}

func (connector *DbConnector) GetMetricRetention(metric string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	key := getMetricRetentionDbKey(metric)
	retention, err := redis.Int64(c.Do("GET", key))
	if err != nil {
		if err != redis.ErrNil {
			return 60, nil
		}
		return 0, fmt.Errorf("Failed GET metric-retention:%s, error: %s", metric, err.Error())
	}
	return retention, nil
}

func (connector *DbConnector) AddTriggerToCheck(triggerId string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SADD", "moira-triggers-tocheck", triggerId)
	if err != nil {
		return fmt.Errorf("Failed to SADD triggers-tocheck triggerID: %s, error: %s", triggerId, err.Error())
	}
	return nil
}

func (connector *DbConnector) GetTriggerToCheck() (*string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerId, err := redis.String(c.Do("SPOP", "moira-triggers-tocheck"))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to SPOP triggers-tocheck, error: %s", err.Error())
	}
	return &triggerId, nil
}

func (connector *DbConnector) AddPatternMetric(pattern, metric string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SADD", fmt.Sprintf("moira-pattern-metrics:%s", pattern), metric)
	if err != nil {
		return fmt.Errorf("Failed to SADD pattern-metrics, pattern: %s, metric: %s, error: %s", pattern, metric, err.Error())
	}
	return err
}

func (connector *DbConnector) SubscribeMetricEvents(tomb *tomb.Tomb) <-chan *moira.MetricEvent {
	c := connector.pool.Get()
	psc := redis.PubSubConn{Conn: c}
	psc.Subscribe("metric-event")

	metricsChannel := make(chan *moira.MetricEvent, 100)
	dataChannel := connector.manageSubscriptions(psc)

	go func() {
		defer c.Close()
		<-tomb.Dying()
		connector.logger.Infof("Calling shutdown, unsubscribe from 'moira-event' redis channel...")
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

func (connector *DbConnector) GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*moira.MetricValue, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, metric := range metrics {
		c.Send("ZRANGEBYSCORE", getMetricDbKey(metric), from, until, "WITHSCORES")
	}
	resultByMetrics, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	res := make(map[string][]*moira.MetricValue)

	for i, resultByMetric := range resultByMetrics {
		metric := metrics[i]
		resultByMetricArr, err := redis.Values(resultByMetric, nil)
		if err != nil {
			return nil, err
		}
		metricsValues := make([]*moira.MetricValue, 0, len(resultByMetricArr)/2)

		for i := 0; i < len(resultByMetricArr); i += 2 {
			val, err := redis.String(resultByMetricArr[i], nil)
			if err != nil {
				return nil, err
			}
			valuesArr := strings.Split(val, " ")
			if len(valuesArr) != 2 {
				return nil, fmt.Errorf("Value format is not valid: %s", val)
			}
			timestampStr, err := redis.String(valuesArr[0], nil)
			if err != nil {
				return nil, err
			}
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				return nil, err
			}
			valueStr, err := redis.String(valuesArr[1], nil)
			if err != nil {
				return nil, err
			}
			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return nil, err
			}
			retentionTimestamp, err := redis.Int64(resultByMetricArr[i+1], nil)
			if err != nil {
				return nil, err
			}
			metricValue := moira.MetricValue{
				RetentionTimestamp: retentionTimestamp,
				Timestamp:          timestamp,
				Value:              value,
			}
			metricsValues = append(metricsValues, &metricValue)
		}
		res[metric] = metricsValues
	}
	return res, nil
}

func (connector *DbConnector) CleanupMetricValues(metric string, toTime int64) error {
	c := connector.pool.Get()
	defer c.Close()

	_, err := c.Do("ZREMRANGEBYSCORE", getMetricDbKey(metric), "-inf", toTime)
	if err != nil {
		return fmt.Errorf("Failed to ZREMRANGEBYSCORE: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) manageSubscriptions(psc redis.PubSubConn) <-chan []byte {
	dataChan := make(chan []byte)
	go func() {
		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				dataChan <- n.Data
			case redis.Subscription:
				if n.Kind == "subscribe" {
					connector.logger.Infof("Subscribe to %s channel, current subscriptions is %v", n.Channel, n.Count)
				} else if n.Kind == "unsubscribe" {
					connector.logger.Infof("Unsubscribe from %s channel, current subscriptions is %v", n.Channel, n.Count)
					if n.Count == 0 {
						connector.logger.Infof("No more subscriptions, exit...")
						close(dataChan)
						return
					}
				}
			}
		}
	}()
	return dataChan
}

func leftJoin(left, right []string) []string {
	rightValues := make(map[string]bool)
	for _, value := range right {
		rightValues[value] = true
	}
	arr := make([]string, 0)
	for _, leftValue := range left {
		if _, ok := rightValues[leftValue]; !ok {
			arr = append(arr, leftValue)
		}
	}
	return arr
}
