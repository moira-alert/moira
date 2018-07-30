package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira/database"
)

func (connector *DbConnector) AddTriggersToCheck(triggerIDs []string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("SADD", triggerToCheckKey, triggerID)
	}
	_, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("failed to add triggers to check: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) GetTriggerToCheck() (string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerID, err := redis.String(c.Do("SPOP", triggerToCheckKey))
	if err != nil {
		if err == redis.ErrNil {
			return "", database.ErrNil
		}
		return "", fmt.Errorf("failed to pop trigger to check: %s", err.Error())
	}
	return triggerID, err
}

var triggerToCheckKey = "moira-triggers-to-check"
