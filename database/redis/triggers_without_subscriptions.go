package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira"
)

// AddTriggersWithoutSubscriptions adds trigger IDs without subscriptions to Redis set
func (connector *DbConnector) AddTriggersWithoutSubscriptions(triggerIDs []string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("SADD", triggersWithoutSubscriptionsKey, triggerID)
	}
	_, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("failed to add triggers without subscription: %s", err.Error())
	}
	return nil
}

// GetTriggersWithoutSubscriptions returns all trigger IDs without subscriptions
func (connector *DbConnector) GetTriggersWithoutSubscriptions() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIds, err := redis.Strings(c.Do("SMEMBERS", triggersWithoutSubscriptionsKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get all triggers without subscription: %s", err.Error())
	}
	return triggerIds, nil
}

// RemoveTriggersWithoutSubscriptions removes trigger IDs without subscriptions from Redis set
func (connector *DbConnector) RemoveTriggersWithoutSubscriptions(triggerIDs []string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("SREM", triggersWithoutSubscriptionsKey, triggerID)
	}
	_, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return fmt.Errorf("failed to add triggers without subscription: %s", err.Error())
	}

	return nil
}

func (connector *DbConnector) updateTriggersWithoutSubscription(newTriggers, oldTriggers []*moira.Trigger) error {
	triggerIDsWithoutSubscription := make([]string, 0)
	triggerIDsWithSubscription := make([]string, 0)

	triggersNotInNewList := moira.LeftJoinTriggers(oldTriggers, newTriggers)
	for _, trigger := range triggersNotInNewList {
		ok, err := connector.triggerHasSubscriptions(trigger)
		if err != nil {
			return err
		}
		if !ok {
			triggerIDsWithoutSubscription = append(triggerIDsWithoutSubscription, trigger.ID)
		}
	}

	for _, trigger := range newTriggers {
		triggerIDsWithSubscription = append(triggerIDsWithSubscription, trigger.ID)
	}

	if len(triggerIDsWithoutSubscription) > 0 {
		err := connector.AddTriggersWithoutSubscriptions(triggerIDsWithoutSubscription)
		if err != nil {
			return err
		}
	}

	if len(triggerIDsWithSubscription) > 0 {
		return connector.RemoveTriggersWithoutSubscriptions(triggerIDsWithSubscription)
	}

	return nil
}

var triggersWithoutSubscriptionsKey = "moira-triggers-without-subscription"
