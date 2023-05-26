package redis

import (
	"fmt"

	"github.com/go-redis/redis/v8"
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

func (connector *DbConnector) AddVMSelectTriggersToCheck(triggerIDs []string) error {
	return connector.addTriggersToCheck(vmselectTriggersToCheckKey, triggerIDs)
}

// GetLocalTriggersToCheck return random trigger ID from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(localTriggersToCheckKey, count)
}

// GetRemoteTriggersToCheck return random remote trigger ID from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(remoteTriggersToCheckKey, count)
}

func (connector *DbConnector) GetVMSelectTriggersToCheck(count int) ([]string, error) {
	return connector.getTriggersToCheck(vmselectTriggersToCheckKey, count)
}

// GetLocalTriggersToCheckCount return number of triggers ID to check from Redis Set
func (connector *DbConnector) GetLocalTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(localTriggersToCheckKey)
}

// GetRemoteTriggersToCheckCount return number of remote triggers ID to check from Redis Set
func (connector *DbConnector) GetRemoteTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(remoteTriggersToCheckKey)
}

func (connector *DbConnector) GetVMSelectTriggersToCheckCount() (int64, error) {
	return connector.getTriggersToCheckCount(vmselectTriggersToCheckKey)
}

func (connector *DbConnector) addTriggersToCheck(key string, triggerIDs []string) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	for _, triggerID := range triggerIDs {
		pipe.SAdd(ctx, key, triggerID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add triggers to check: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) getTriggersToCheck(key string, count int) ([]string, error) {
	ctx := connector.context
	c := *connector.client

	triggerIDs, err := c.SPopN(ctx, key, int64(count)).Result()
	if err != nil {
		if err == redis.Nil {
			return make([]string, 0), database.ErrNil
		}
		return make([]string, 0), fmt.Errorf("failed to pop trigger to check: %s", err.Error())
	}
	return triggerIDs, err
}

func (connector *DbConnector) getTriggersToCheckCount(key string) (int64, error) {
	ctx := connector.context
	c := *connector.client

	triggersToCheckCount, err := c.SCard(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get trigger to check count: %s", err.Error())
	}
	return triggersToCheckCount, nil
}

var remoteTriggersToCheckKey = "moira-remote-triggers-to-check"
var vmselectTriggersToCheckKey = "moira-vmselect-triggers-to-check"
var localTriggersToCheckKey = "moira-triggers-to-check"
