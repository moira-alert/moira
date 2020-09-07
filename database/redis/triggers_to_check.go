package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/database"
)

// AddLocalTriggersToCheck gets trigger IDs and save it to Redis Set
func (connector *DbConnector) AddLocalTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(localTriggersToCheckKey, triggerIDs)
}

// AddRemoteTriggersToCheck gets remote trigger IDs and save it to Redis Set
func (connector *DbConnector) AddRemoteTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(remoteTriggersToCheckKey, triggerIDs)
}

// GetLocalTriggersToCheck return random trigger ID from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(localTriggersToCheckKey, count)
}

// GetRemoteTriggersToCheck return random remote trigger ID from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(remoteTriggersToCheckKey, count)
}

// GetLocalTriggersToCheckCount return number of triggers ID to check from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(localTriggersToCheckKey)
}

// GetRemoteTriggersToCheckCount return number of remote triggers ID to check from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(remoteTriggersToCheckKey)
}

func (connector *DbConnector) addTriggersToCheck(key string, triggerIDs []string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("SADD", key, triggerID)
	}
	_, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("failed to add triggers to check: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) getTriggersToCheck(key string, count int) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIDs, err := redis.Strings(c.Do("SPOP", key, count))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), database.ErrNil
		}
		return make([]string, 0), fmt.Errorf("failed to pop trigger to check: %s", err.Error())
	}
	return triggerIDs, err
}

func (connector *DbConnector) getTriggersToCheckCount(key string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggersToCheckCount, err := redis.Int64(c.Do("SCARD", key))
	if err != nil {
		if err == redis.ErrNil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get trigger to check count: %s", err.Error())
	}
	return triggersToCheckCount, nil
}

var remoteTriggersToCheckKey = "moira-remote-triggers-to-check"
var localTriggersToCheckKey = "moira-triggers-to-check"
