package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/database"
)

// AddRemoteTriggersToCheck gets remote trigger IDs and save it to Redis Set
func (connector *DbConnector) AddRemoteTriggersToCheck(triggerIDs []string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("SADD", remoteTriggersToCheckKey, triggerID)
	}
	_, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("failed to add remote triggers to check: %s", err.Error())
	}
	return nil
}

// GetRemoteTriggerToCheck return random remote trigger ID from Redis Set
func (connector *DbConnector) GetRemoteTriggerToCheck() (string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerID, err := redis.String(c.Do("SPOP", remoteTriggersToCheckKey))
	if err != nil {
		if err == redis.ErrNil {
			return "", database.ErrNil
		}
		return "", fmt.Errorf("failed to pop remote trigger to check: %s", err.Error())
	}
	return triggerID, err
}

// GetRemoteTriggersToCheckCount return number of remote triggers ID to check from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheckCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggersToCheckCount, err := redis.Int64(c.Do("SCARD", remoteTriggersToCheckKey))
	if err != nil {
		if err == redis.ErrNil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get trigger to check count: %s", err.Error())
	}
	return triggersToCheckCount, nil
}

var remoteTriggersToCheckKey = "moira-remote-triggers-to-check"
