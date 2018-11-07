package redis

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetSubscription returns subscription data by given id, if no value, return database.ErrNil error
func (connector *DbConnector) GetSubscription(id string) (moira.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	subscription, err := reply.Subscription(c.Do("GET", subscriptionKey(id)))
	if err != nil {
		return subscription, err
	}
	subscription.ID = id
	return subscription, nil
}

// GetSubscriptions returns subscriptions data by given ids, len of subscriptionIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetSubscriptions(subscriptionIDs []string) ([]*moira.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, id := range subscriptionIDs {
		c.Send("GET", subscriptionKey(id))
	}
	subscriptions, err := reply.Subscriptions(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	for i := range subscriptions {
		if subscriptions[i] != nil {
			subscriptions[i].ID = subscriptionIDs[i]
		}
	}
	return subscriptions, nil
}

// SaveSubscription writes subscription data, updates tags subscriptions and user subscriptions
func (connector *DbConnector) SaveSubscription(subscription *moira.SubscriptionData) error {
	oldSubscription, getSubError := connector.GetSubscription(subscription.ID)
	if getSubError != nil && getSubError != database.ErrNil {
		return getSubError
	}
	oldTriggers, err := connector.getSubscriptionTriggers(&oldSubscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	if getSubError != database.ErrNil {
		addSendSubscriptionRequest(c, subscription, &oldSubscription)
	} else {
		addSendSubscriptionRequest(c, subscription, nil)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	newTriggers, err := connector.getSubscriptionTriggers(subscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	return connector.refreshUnusedTriggers(newTriggers, oldTriggers)
}

// SaveSubscriptions writes subscriptions, updates tags subscriptions and user subscriptions
func (connector *DbConnector) SaveSubscriptions(subscriptions []*moira.SubscriptionData) error {
	ids := make([]string, len(subscriptions))
	for i, subscription := range subscriptions {
		ids[i] = subscription.ID
	}
	oldSubscriptions, err := connector.GetSubscriptions(ids)
	if err != nil {
		return err
	}
	oldTriggers, err := connector.getSubscriptionsTriggers(oldSubscriptions)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for i, subscription := range subscriptions {
		addSendSubscriptionRequest(c, subscription, oldSubscriptions[i])
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	newTriggers, err := connector.getSubscriptionsTriggers(subscriptions)
	if err != nil {
		return err
	}
	if err := connector.refreshUnusedTriggers(newTriggers, oldTriggers); err != nil {
		return err
	}

	return nil
}

// RemoveSubscription deletes subscription data and removes subscriptionID from users and tags subscriptions
func (connector *DbConnector) RemoveSubscription(subscriptionID string) error {
	subscription, err := connector.GetSubscription(subscriptionID)
	if err != nil {
		if err == database.ErrNil {
			return nil
		}
		return err
	}
	oldTriggers, err := connector.getSubscriptionTriggers(&subscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SREM", userSubscriptionsKey(subscription.User), subscriptionID)
	for _, tag := range subscription.Tags {
		c.Send("SREM", tagSubscriptionKey(tag), subscriptionID)
	}
	c.Send("DEL", subscriptionKey(subscription.ID))
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	err = connector.refreshUnusedTriggers([]*moira.Trigger{}, oldTriggers)
	if err != nil {
		return fmt.Errorf("failed to update triggers by subscription: %s", err.Error())
	}
	return nil
}

// GetUserSubscriptionIDs returns subscriptions ids by given login
func (connector *DbConnector) GetUserSubscriptionIDs(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	subscriptions, err := redis.Strings(c.Do("SMEMBERS", userSubscriptionsKey(login)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	return subscriptions, nil
}

// GetTagsSubscriptions gets all subscriptionsIDs by given tag list and read subscriptions.
// Len of subscriptionIDs is equal to len of returned values array. If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTagsSubscriptions(tags []string) ([]*moira.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	tagKeys := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		tagKeys = append(tagKeys, fmt.Sprintf("moira-tag-subscriptions:%s", tag))
	}
	values, err := redis.Values(c.Do("SUNION", tagKeys...))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for tags %v: %s", tags, err.Error())
	}
	var subscriptionsIDs []string
	if err = redis.ScanSlice(values, &subscriptionsIDs); err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for tags %v: %s", tags, err.Error())
	}
	if len(subscriptionsIDs) == 0 {
		return make([]*moira.SubscriptionData, 0), nil
	}

	subscriptionsData, err := connector.GetSubscriptions(subscriptionsIDs)
	if err != nil {
		return nil, err
	}
	return subscriptionsData, nil
}

func addSendSubscriptionRequest(c redis.Conn, subscription *moira.SubscriptionData, oldSubscription *moira.SubscriptionData) error {
	bytes, err := json.Marshal(subscription)
	if err != nil {
		return err
	}
	if oldSubscription != nil {
		for _, tag := range oldSubscription.Tags {
			c.Send("SREM", tagSubscriptionKey(tag), subscription.ID)
		}
		if oldSubscription.User != subscription.User {
			c.Send("SREM", userSubscriptionsKey(oldSubscription.User), subscription.ID)
		}
	}
	for _, tag := range subscription.Tags {
		c.Send("SADD", tagSubscriptionKey(tag), subscription.ID)
	}
	c.Send("SADD", userSubscriptionsKey(subscription.User), subscription.ID)
	c.Send("SET", subscriptionKey(subscription.ID), bytes)
	return nil
}

func (connector *DbConnector) getSubscriptionTriggers(subscription *moira.SubscriptionData) ([]*moira.Trigger, error) {
	if subscription == nil || len(subscription.Tags) == 0 {
		return make([]*moira.Trigger, 0), nil
	}

	c := connector.pool.Get()
	defer c.Close()

	tagKeys := make([]interface{}, 0, len(subscription.Tags))
	for _, tag := range subscription.Tags {
		tagKeys = append(tagKeys, tagTriggersKey(tag))
	}

	values, err := redis.Values(c.Do("SINTER", tagKeys...))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve triggers for tags %v: %s", subscription.Tags, err.Error())
	}

	var triggerIDs []string
	if err = redis.ScanSlice(values, &triggerIDs); err != nil {
		return nil, fmt.Errorf("failed to retrieve triggers for tags %v: %s", subscription.Tags, err.Error())
	}
	if len(triggerIDs) == 0 {
		return make([]*moira.Trigger, 0), nil
	}
	return connector.GetTriggers(triggerIDs)
}

func (connector *DbConnector) getSubscriptionsTriggers(subscriptions []*moira.SubscriptionData) ([]*moira.Trigger, error) {
	triggersMap := make(map[string]*moira.Trigger)
	triggers := make([]*moira.Trigger, 0)

	for _, subscription := range subscriptions {
		triggersBySubscription, err := connector.getSubscriptionTriggers(subscription)
		if err != nil {
			return triggers, err
		}
		for _, trigger := range triggersBySubscription {
			triggersMap[trigger.ID] = trigger
		}
	}
	for _, trigger := range triggersMap {
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

func subscriptionKey(id string) string {
	return fmt.Sprintf("moira-subscription:%s", id)
}

func userSubscriptionsKey(userName string) string {
	return fmt.Sprintf("moira-user-subscriptions:%s", userName)
}
