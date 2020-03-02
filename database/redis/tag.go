package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	tagNames, err := redis.Strings(c.Do("SMEMBERS", tagsKey))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tags: %s", err.Error())
	}
	return tagNames, nil
}

// RemoveTag deletes tag from tags list, deletes triggerIDs and subscriptionsIDs lists by given tag
func (connector *DbConnector) RemoveTag(tagName string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SREM", tagsKey, tagName)
	c.Send("DEL", tagSubscriptionKey(tagName))
	c.Send("DEL", tagTriggersKey(tagName))
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetTagTriggerIDs gets all triggersIDs by given tagName
func (connector *DbConnector) GetTagTriggerIDs(tagName string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIDs, err := redis.Strings(c.Do("SMEMBERS", tagTriggersKey(tagName)))
	if err != nil {
		if err == redis.ErrNil {
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
