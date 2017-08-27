package redis

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

// GetSubscription returns subscription data by given id
func (connector *DbConnector) GetSubscription(id string) (moira.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	subscription, err := reply.Subscription(c.Do("GET", moiraSubscription(id)))
	if err != nil {
		connector.metrics.SubsMalformed.Mark(1)
		if err != database.ErrNil {
			return subscription, fmt.Errorf("Failed to get subscription data for id %s: %s", id, err.Error())
		}
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
		c.Send("GET", moiraSubscription(id))
	}
	subscriptions, err := reply.Subscriptions(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	for i := range subscriptions {
		subscriptions[i].ID = subscriptionIDs[i]
	}
	return subscriptions, nil
}

//WriteSubscriptions writes subscriptions data
func (connector *DbConnector) WriteSubscriptions(subscriptions []*moira.SubscriptionData) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	subscriptionsBytes := make([][]byte, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		bytes, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		subscriptionsBytes = append(subscriptionsBytes, bytes)
		c.Send("SET", moiraSubscription(subscription.ID), bytes)
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//SaveSubscription writes subscription data, updates tags subscriptions and user subscriptions
func (connector *DbConnector) SaveSubscription(subscription *moira.SubscriptionData) error {
	oldSubscription, err := connector.GetSubscription(subscription.ID)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, tag := range oldSubscription.Tags {
		c.Send("SREM", moiraTagSubscription(tag), subscription.ID)
	}
	for _, tag := range subscription.Tags {
		c.Send("SADD", moiraTagSubscription(tag), subscription.ID)
	}
	c.Send("SADD", moiraUserSubscriptions(subscription.User), subscription.ID)
	c.Send("SET", moiraSubscription(subscription.ID), bytes)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//RemoveSubscription deletes subscription data and removes subscriptionID from users and tags subscriptions
func (connector *DbConnector) RemoveSubscription(subscriptionID string, userLogin string) error {
	subscription, err := connector.GetSubscription(subscriptionID)
	if err != nil {
		return nil
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SREM", moiraUserSubscriptions(userLogin), subscriptionID)
	for _, tag := range subscription.Tags {
		c.Send("SREM", moiraTagSubscription(tag), subscriptionID)
	}
	c.Send("DEL", moiraSubscription(subscription.ID))
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//GetUserSubscriptionIDs returns subscriptions ids by given login
func (connector *DbConnector) GetUserSubscriptionIDs(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	subscriptions, err := redis.Strings(c.Do("SMEMBERS", moiraUserSubscriptions(login)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	return subscriptions, nil
}

func moiraSubscription(id string) string {
	return fmt.Sprintf("moira-subscription:%s", id)
}

func moiraTagSubscription(tag string) string {
	return fmt.Sprintf("moira-tag-subscriptions:%s", tag)
}

func moiraUserSubscriptions(userName string) string {
	return fmt.Sprintf("moira-user-subscriptions:%s", userName)
}
