package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetSubscription returns subscription data by given id, if no value, return database.ErrNil error
func (connector *DbConnector) GetSubscription(id string) (moira.SubscriptionData, error) {
	c := *connector.client

	subscription, err := reply.Subscription(c.Get(connector.context, subscriptionKey(id)))
	if err != nil {
		return subscription, err
	}
	if subscription.Tags == nil {
		subscription.Tags = []string{}
	}
	subscription.ID = id
	return subscription, nil
}

// GetSubscriptions returns subscriptions data by given ids, len of subscriptionIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetSubscriptions(subscriptionIDs []string) ([]*moira.SubscriptionData, error) {
	c := *connector.client
	subscriptions := make([]*moira.SubscriptionData, 0, len(subscriptionIDs))
	results := make([]*redis.StringCmd, 0, len(subscriptionIDs))

	pipe := c.TxPipeline()
	for _, id := range subscriptionIDs {
		result := pipe.Get(connector.context, subscriptionKey(id))
		results = append(results, result)
	}
	_, err := pipe.Exec(connector.context)

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	for i, result := range results {
		if result.Val() != "" {
			subscription, err := reply.Subscription(result)
			if err != nil {
				return nil, err
			}
			subscription.ID = subscriptionIDs[i]
			if subscription.Tags == nil {
				subscription.Tags = []string{}
			}
			subscriptions = append(subscriptions, &subscription)
		} else {
			subscriptions = append(subscriptions, nil)
		}
	}
	return subscriptions, nil
}

// SaveSubscription writes subscription data, updates tags subscriptions and user subscriptions
func (connector *DbConnector) SaveSubscription(subscription *moira.SubscriptionData) error {
	var oldSubscription *moira.SubscriptionData

	if subscription, err := connector.GetSubscription(subscription.ID); err == nil {
		oldSubscription = &subscription
	} else if err != database.ErrNil {
		return err
	}
	oldTriggers, err := connector.getSubscriptionTriggers(oldSubscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	if err = connector.updateSubscription(subscription, oldSubscription); err != nil {
		return fmt.Errorf("failed to update subscription: %s", err.Error())
	}
	newTriggers, err := connector.getSubscriptionTriggers(subscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	return connector.refreshUnusedTriggers(newTriggers, oldTriggers)
}

func (connector *DbConnector) updateSubscription(newSubscription *moira.SubscriptionData, oldSubscription *moira.SubscriptionData) error {
	c := *connector.client

	pipe := c.TxPipeline()                                                                 //nolint
	addSendSubscriptionRequest(connector.context, pipe, *newSubscription, oldSubscription) //nolint
	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// SaveSubscriptions writes subscriptions, updates tags subscriptions and user subscriptions
func (connector *DbConnector) SaveSubscriptions(newSubscriptions []*moira.SubscriptionData) error {
	ids := make([]string, len(newSubscriptions))
	for i, subscription := range newSubscriptions {
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
	if err = connector.updateSubscriptions(oldSubscriptions, newSubscriptions); err != nil {
		return err
	}
	newTriggers, err := connector.getSubscriptionsTriggers(newSubscriptions)
	if err != nil {
		return err
	}
	if err := connector.refreshUnusedTriggers(newTriggers, oldTriggers); err != nil {
		return fmt.Errorf("failed to update triggers by subscription: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) updateSubscriptions(oldSubscriptions []*moira.SubscriptionData, newSubscriptions []*moira.SubscriptionData) error {
	c := *connector.client

	pipe := c.TxPipeline()
	for i, newSubscription := range newSubscriptions {
		addSendSubscriptionRequest(connector.context, pipe, *newSubscription, oldSubscriptions[i]) //nolint
	}
	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
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
	triggers, err := connector.getSubscriptionTriggers(&subscription)
	if err != nil {
		return fmt.Errorf("failed to get triggers by subscription: %s", err.Error())
	}
	if err := connector.removeSubscription(&subscription); err != nil {
		return fmt.Errorf("failed to remove subscription: %s", err.Error())
	}
	if err := connector.refreshUnusedTriggers([]*moira.Trigger{}, triggers); err != nil {
		return fmt.Errorf("failed to update triggers by subscription: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) removeSubscription(subscription *moira.SubscriptionData) error {
	c := *connector.client

	pipe := c.TxPipeline()
	pipe.SRem(connector.context, userSubscriptionsKey(subscription.User), subscription.ID)   //nolint
	pipe.SRem(connector.context, teamSubscriptionsKey(subscription.TeamID), subscription.ID) //nolint
	for _, tag := range subscription.Tags {
		c.SRem(connector.context, tagSubscriptionKey(tag), subscription.ID) //nolint
	}
	pipe.SRem(connector.context, anyTagsSubscriptionsKey, subscription.ID) //nolint
	pipe.Del(connector.context, subscriptionKey(subscription.ID))          //nolint
	if _, err := pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetUserSubscriptionIDs returns subscriptions ids by given login
func (connector *DbConnector) GetUserSubscriptionIDs(login string) ([]string, error) {
	c := *connector.client

	subscriptions, err := c.SMembers(connector.context, userSubscriptionsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	return subscriptions, nil
}

// GetTeamSubscriptionIDs returns subscriptions ids by given team id
func (connector *DbConnector) GetTeamSubscriptionIDs(teamID string) ([]string, error) {
	c := *connector.client

	subscriptions, err := c.SMembers(connector.context, teamSubscriptionsKey(teamID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscriptions for team id %s: %w", teamID, err)
	}
	return subscriptions, nil
}

// GetTagsSubscriptions gets all subscriptionsIDs by given tag list and read subscriptions.
// Len of subscriptionIDs is equal to len of returned values array. If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTagsSubscriptions(tags []string) ([]*moira.SubscriptionData, error) {
	subscriptionsIDs, err := connector.getSubscriptionsIDsByTags(tags)
	if err != nil {
		return nil, err
	}

	if len(subscriptionsIDs) == 0 {
		return make([]*moira.SubscriptionData, 0), nil
	}

	return connector.GetSubscriptions(subscriptionsIDs)
}

func (connector *DbConnector) getSubscriptionsIDsByTags(tags []string) ([]string, error) {
	c := *connector.client

	tagKeys := make([]string, 0, len(tags))

	for _, tag := range tags {
		tagKeys = append(tagKeys, tagSubscriptionKey(tag))
	}
	tagKeys = append(tagKeys, anyTagsSubscriptionsKey)
	values, err := c.SUnion(connector.context, tagKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscriptions for tags %v: %s", tags, err.Error())
	}
	return values, nil
}

func addSendSubscriptionRequest(context context.Context, pipe redis.Pipeliner, subscription moira.SubscriptionData, oldSubscription *moira.SubscriptionData) error {
	if subscription.AnyTags {
		subscription.Tags = nil
	}
	bytes, err := json.Marshal(subscription)
	if err != nil {
		return err
	}
	if oldSubscription != nil {
		for _, tag := range oldSubscription.Tags {
			pipe.SRem(context, tagSubscriptionKey(tag), subscription.ID) //nolint
		}
		if oldSubscription.User != subscription.User {
			pipe.SRem(context, userSubscriptionsKey(oldSubscription.User), subscription.ID) //nolint
		}
		if oldSubscription.TeamID != subscription.TeamID {
			pipe.SRem(context, teamSubscriptionsKey(oldSubscription.TeamID), subscription.ID) //nolint
		}
	}

	for _, tag := range subscription.Tags {
		pipe.SAdd(context, tagSubscriptionKey(tag), subscription.ID) //nolint
	}

	if subscription.AnyTags {
		pipe.SAdd(context, anyTagsSubscriptionsKey, subscription.ID) //nolint
	}

	if subscription.User != "" {
		pipe.SAdd(context, userSubscriptionsKey(subscription.User), subscription.ID) //nolint
	}
	if subscription.TeamID != "" {
		pipe.SAdd(context, teamSubscriptionsKey(subscription.TeamID), subscription.ID) //nolint
	}
	pipe.Set(context, subscriptionKey(subscription.ID), bytes, redis.KeepTTL) //nolint
	return nil
}

func (connector *DbConnector) getTriggersIdsByTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return make([]string, 0), nil
	}

	c := *connector.client

	tagKeys := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagKeys = append(tagKeys, tagTriggersKey(tag))
	}

	values, err := c.SInter(connector.context, tagKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve triggers for tags %v: %s", tags, err.Error())
	}

	return values, nil
}

func (connector *DbConnector) DeleteAllSubscriptions() {
	subs := connector.Client().Keys(context.Background(), subscriptionKey("*"))
	connector.Client().Del(context.Background(), subs.Val()...)

	subs = connector.Client().Keys(context.Background(), userSubscriptionsKey("*"))
	connector.Client().Del(context.Background(), subs.Val()...)

	subs = connector.Client().Keys(context.Background(), teamSubscriptionsKey("*"))
	connector.Client().Del(context.Background(), subs.Val()...)
}

func (connector *DbConnector) getSubscriptionTriggers(subscription *moira.SubscriptionData) ([]*moira.Trigger, error) {
	if subscription == nil {
		return make([]*moira.Trigger, 0), nil
	}
	triggersIDs, err := connector.getTriggersIdsByTags(subscription.Tags)
	if err != nil {
		return nil, err
	}
	if len(triggersIDs) == 0 {
		return make([]*moira.Trigger, 0), nil
	}
	return connector.GetTriggers(triggersIDs)
}

func (connector *DbConnector) getSubscriptionsTriggers(subscriptions []*moira.SubscriptionData) ([]*moira.Trigger, error) {
	triggersMap := make(map[string]*moira.Trigger)
	triggers := make([]*moira.Trigger, 0)

	for _, subscription := range subscriptions {
		subscriptionTriggers, err := connector.getSubscriptionTriggers(subscription)
		if err != nil {
			return triggers, err
		}
		for _, trigger := range subscriptionTriggers {
			if trigger == nil {
				continue
			}
			triggersMap[trigger.ID] = trigger
		}
	}
	for _, trigger := range triggersMap {
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

func subscriptionKey(id string) string {
	return "moira-subscription:" + id
}

func userSubscriptionsKey(userName string) string {
	return "moira-user-subscriptions:" + userName
}

func teamSubscriptionsKey(teamID string) string {
	return fmt.Sprintf("moira-team-subscriptions:%s", teamID)
}

const anyTagsSubscriptionsKey = "{moira-tag-subscriptions}:moira-any-tags-subscriptions"
