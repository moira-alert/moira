package goredis

import (
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/goredis/reply"
)

// GetTriggers returns triggers data by given ids, len of triggerIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTriggers(triggerIDs []string) ([]*moira.Trigger, error) {
	c := *connector.client

	triggerResults := make([]*redis.StringCmd, 0, len(triggerIDs))
	triggerTagsResults := make([]*redis.StringSliceCmd, 0, len(triggerIDs))
	pipe := c.TxPipeline()

	for _, triggerID := range triggerIDs {
		result := pipe.Get(connector.context, triggerKey(triggerID)) //nolint
		triggerResults = append(triggerResults, result)
		tagsResult := pipe.SMembers(connector.context, triggerTagsKey(triggerID)) //nolint
		triggerTagsResults = append(triggerTagsResults, tagsResult)
	}
	_, err := pipe.Exec(connector.context)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	triggers := make([]*moira.Trigger, len(triggerIDs))
	for i := 0; i < len(triggerIDs); i++ {
		trigger, err := connector.getTriggerWithTags(triggerResults[i], triggerTagsResults[i], triggerIDs[i])
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, err
		}
		triggers[i] = &trigger
	}
	return triggers, nil
}

func (connector *DbConnector) getTriggerWithTags(triggerRaw *redis.StringCmd, tagsRaw *redis.StringSliceCmd, triggerID string) (moira.Trigger, error) {
	trigger, err := reply.Trigger(triggerRaw)
	if err != nil {
		return trigger, err
	}
	triggerTags, err := tagsRaw.Result()
	if err != nil {
		connector.logger.Errorf("Error getting trigger tags, id: %s, error: %s", triggerID, err.Error())
	}

	if len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	trigger.ID = triggerID
	return trigger, nil
}

func (connector *DbConnector) triggerHasSubscriptions(trigger *moira.Trigger) (bool, error) {
	if trigger == nil || len(trigger.Tags) == 0 {
		return false, nil
	}
	subscriptions, err := connector.GetTagsSubscriptions(trigger.Tags)
	if err != nil {
		return false, err
	}

	for _, subscription := range subscriptions {
		if subscription == nil {
			continue
		}
		if subscription.AnyTags || moira.Subset(subscription.Tags, trigger.Tags) {
			return true, nil
		}
	}

	return false, nil
}

func triggerKey(triggerID string) string {
	return "moira-trigger:" + triggerID
}

func triggerTagsKey(triggerID string) string {
	return "moira-trigger-tags:" + triggerID
}
