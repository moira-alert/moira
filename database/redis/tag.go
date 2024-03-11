package redis

import (
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := *connector.client

	tagNames, err := c.SMembers(connector.context, tagsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tags: %w", err)
	}

	return tagNames, nil
}

// CreateTags creates an array of tags without binding to a subscription or trigger
func (connector *DbConnector) CreateTags(tags []string) error {
	pipe := (*connector.client).TxPipeline()
	ctx := connector.context

	for _, tag := range tags {
		pipe.SAdd(ctx, tagsKey, tag)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}

	return nil
}

// RemoveTag deletes tag from tags list, deletes triggerIDs and subscriptionsIDs lists by given tag
func (connector *DbConnector) RemoveTag(tagName string) error {
	pipe := (*connector.client).TxPipeline()
	pipe.SRem(connector.context, tagsKey, tagName)
	pipe.Del(connector.context, tagSubscriptionKey(tagName))
	pipe.Del(connector.context, tagTriggersKey(tagName))

	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %w", err)
	}

	return nil
}

// GetTagTriggerIDs gets all triggersIDs by given tagName
func (connector *DbConnector) GetTagTriggerIDs(tagName string) ([]string, error) {
	c := *connector.client

	triggerIDs, err := c.SMembers(connector.context, tagTriggersKey(tagName)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return make([]string, 0), nil
		}

		return nil, fmt.Errorf("failed to retrieve tag triggers: %s, err: %w", tagName, err)
	}

	return triggerIDs, nil
}

// CleanUpAbandonedTags deletes tags for which triggers and subscriptions don't exist.
// Returns count of deleted tags.
func (connector *DbConnector) CleanUpAbandonedTags() (int, error) {
	var count int
	client := *connector.client

	iter := client.SScan(connector.context, tagsKey, 0, "*", 0).Iterator()
	for iter.Next(connector.context) {
		tag := iter.Val()

		result, err := client.Exists(connector.context, tagTriggersKey(tag)).Result()
		if err != nil {
			return count, fmt.Errorf("failed to check tag triggers existence, error: %w", err)
		}

		if isTriggerExists := result == 1; isTriggerExists {
			continue
		}

		result, err = client.Exists(connector.context, tagSubscriptionKey(tag)).Result()
		if err != nil {
			return count, fmt.Errorf("failed to check tag subscription existence, error: %w", err)
		}

		if isSubscriptionExists := result == 1; isSubscriptionExists {
			continue
		}

		err = client.SRem(connector.context, tagsKey, tag).Err()
		if err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

var tagsKey = "moira-tags"

func tagTriggersKey(tagName string) string {
	return "{moira-tag-triggers}:" + tagName
}

func tagSubscriptionKey(tagName string) string {
	return "{moira-tag-subscriptions}:" + tagName
}
