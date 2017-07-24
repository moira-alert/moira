package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
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
		triggers = append(triggers, toTrigger(triggerSE))
	}

	return triggers, nil
}

func (connector *DbConnector) GetPatternMetrics(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	metrics, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("moira-pattern-metrics:%s", pattern)))
	if err != nil {
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
