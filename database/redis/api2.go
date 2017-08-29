package redis

import (
	"encoding/json"
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
		c.Do("SADD", moiraPatternsList, pattern)
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
