package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
)

func (connector *DbConnector) AddTriggerToCheck(triggerID string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SADD", "moira-triggers-tocheck", triggerID)
	if err != nil {
		return fmt.Errorf("Failed to SADD triggers-tocheck triggerID: %s, error: %s", triggerID, err.Error())
	}
	return nil
}

func (connector *DbConnector) GetTriggerToCheck() (*string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerID, err := redis.String(c.Do("SPOP", "moira-triggers-tocheck"))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to SPOP triggers-tocheck, error: %s", err.Error())
	}
	return &triggerID, nil
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
