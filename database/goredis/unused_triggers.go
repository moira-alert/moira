package goredis

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

// MarkTriggersAsUnused adds unused trigger IDs to Redis set
func (connector *DbConnector) MarkTriggersAsUnused(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}

	c := *connector.client
	pipe := c.TxPipeline()

	for _, triggerID := range triggerIDs {
		pipe.SAdd(connector.context, unusedTriggersKey, triggerID) //nolint
	}
	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to mark triggers as unused: %s", err.Error())
	}
	return nil
}

// GetUnusedTriggerIDs returns all unused trigger IDs
func (connector *DbConnector) GetUnusedTriggerIDs() ([]string, error) {
	ctx := connector.context
	c := *connector.client

	triggerIds, err := c.SMembers(ctx, unusedTriggersKey).Result()
	if err != nil {
		if err == redis.Nil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("failed to get all unused triggers: %s", err.Error())
	}
	return triggerIds, nil
}

// MarkTriggersAsUsed removes trigger IDs from Redis set
func (connector *DbConnector) MarkTriggersAsUsed(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}

	c := *connector.client
	pipe := c.TxPipeline()
	for _, triggerID := range triggerIDs {
		pipe.SRem(connector.context, unusedTriggersKey, triggerID) //nolint
	}
	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to mark triggers as used: %s", err.Error())
	}

	return nil
}

// refreshUnusedTriggers gets two triggers lists: newTriggers and oldTriggers
// It filters triggers which are presented in oldTriggers but not in newTriggers.
// For every trigger in that diff-list it checks whether this trigger has any subscription and mark it unused if not.
// At the end, refreshUnusedTriggers mark all newTriggers as used
func (connector *DbConnector) refreshUnusedTriggers(newTriggers, oldTriggers []*moira.Trigger) error {
	unusedTriggerIDs := make([]string, 0)
	usedTriggerIDs := make([]string, 0)

	triggersNotInNewList := moira.GetTriggerListsDiff(oldTriggers, newTriggers)
	for _, trigger := range triggersNotInNewList {
		ok, err := connector.triggerHasSubscriptions(trigger)
		if err != nil {
			return err
		}
		if !ok {
			unusedTriggerIDs = append(unusedTriggerIDs, trigger.ID)
		}
	}

	for _, trigger := range newTriggers {
		if trigger != nil {
			usedTriggerIDs = append(usedTriggerIDs, trigger.ID)
		}
	}

	if len(unusedTriggerIDs) > 0 {
		err := connector.MarkTriggersAsUnused(unusedTriggerIDs...)
		if err != nil {
			return err
		}
	}

	if len(usedTriggerIDs) > 0 {
		return connector.MarkTriggersAsUsed(usedTriggerIDs...)
	}

	return nil
}

var unusedTriggersKey = "moira-unused-triggers"
