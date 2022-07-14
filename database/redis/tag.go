package redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := *connector.client
	tagNames, err := c.SMembers(connector.context, tagsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tags: %s", err.Error())
	}
	return tagNames, nil
}

// RemoveTag deletes tag from tags list, deletes triggerIDs and subscriptionsIDs lists by given tag
func (connector *DbConnector) RemoveTag(tagName string) error {
	pipe := (*connector.client).TxPipeline()
	pipe.SRem(connector.context, tagsKey, tagName)
	pipe.Del(connector.context, tagSubscriptionKey(tagName))
	pipe.Del(connector.context, tagTriggersKey(tagName))

	_, err := pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// AddTag adds tag to database.
func (connector *DbConnector) AddTag(tagName string) error {
	client := *connector.client
	err := client.SAdd(connector.context, tagsKey, tagName).Err()

	if err != nil {
		return fmt.Errorf("failed to add tag: %s", err.Error())
	}
	return nil
}

// GetTagTriggerIDs gets all triggersIDs by given tagName
func (connector *DbConnector) GetTagTriggerIDs(tagName string) ([]string, error) {
	c := *connector.client

	triggerIDs, err := c.SMembers(connector.context, tagTriggersKey(tagName)).Result()
	if err != nil {
		if err == redis.Nil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("failed to retrieve tag triggers:%s, err: %s", tagName, err.Error())
	}
	return triggerIDs, nil
}

// CleanUpAbandonedTags deletes tags for which triggers don't exist.
// Returns count of deleted tags.
func (connector *DbConnector) CleanUpAbandonedTags() (int, error) {
	counter := 0
	client := *connector.client

	switch c := client.(type) {
	case *redis.ClusterClient:
		err := c.ForEachMaster(connector.context, func(ctx context.Context, shard *redis.Client) error {
			count, err := cleanUpAbandonedTagsOnRedisNode(connector, shard)
			if err != nil {
				return err
			}
			counter += count

			return nil
		})
		if err != nil {
			return 0, err
		}
	default:
		count, err := cleanUpAbandonedTagsOnRedisNode(connector, c)
		if err != nil {
			return 0, err
		}
		counter += count
	}

	return counter, nil
}

func cleanUpAbandonedTagsOnRedisNode(connector *DbConnector, client redis.UniversalClient) (int, error) {
	var count int
	iter := client.SScan(connector.context, tagsKey, 0, "*", 0).Iterator()
	for iter.Next(connector.context) {
		tag := iter.Val()

		result, err := (*connector.client).Exists(connector.context, tagTriggersKey(tag)).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to check tag triggers existence, error: %v", err)
		}
		if isTriggerExists := result == 1; !isTriggerExists {
			err = connector.RemoveTag(tag)
			client.SRem(connector.context, tagsKey, tag)

			if err != nil {
				return 0, err
			}
			count++
		}
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
