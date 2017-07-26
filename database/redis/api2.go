package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
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
	//todo прибраmься/разбить на 2 метода
	now := time.Now().Unix()

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SET", fmt.Sprintf("moira-notifier-throttling-beginning:%s", triggerId), now)
	c.Send("DEL", fmt.Sprintf("moira-notifier-next:%s", triggerId))
	c.Send("ZRANGEBYSCORE", "moira-notifier-notifications", "-inf", "+inf")
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	notificationStrings, err := redis.ByteSlices(rawResponse[2], nil)
	if err != nil {
		return err
	}
	notifications, err := connector.convertNotifications(rawResponse[2])
	if err != nil {
		return err
	}
	c.Send("MULTI")
	for i, notification := range notifications {
		if notification.Event.TriggerID == triggerId {
			c.Send("ZADD", "moira-notifier-notifications", now, notificationStrings[i])
		}
	}
	_, err = c.Do("EXEC")
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
