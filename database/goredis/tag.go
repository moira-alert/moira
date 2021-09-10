package goredis

import (
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

var tagsKey = "moira-tags"

func tagTriggersKey(tagName string) string {
	return "moira-tag-triggers:" + tagName
}

func tagSubscriptionKey(tagName string) string {
	return "moira-tag-subscriptions:" + tagName
}
