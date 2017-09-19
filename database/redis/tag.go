package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := connector.pool.Get()
	if c.Err() != nil {
		return nil, c.Err()
	}
	if c.Err() != nil {
		return nil, c.Err()
	}
	defer c.Close()

	tagNames, err := redis.Strings(c.Do("SMEMBERS", tagsKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags: %s", err.Error())
	}
	return tagNames, nil
}

// RemoveTag deletes tag from tags list, deletes triggerIDs and subscriptionsIDs lists by given tag
func (connector *DbConnector) RemoveTag(tagName string) error {
	c := connector.pool.Get()
	if c.Err() != nil {
		return c.Err()
	}
	if c.Err() != nil {
		return c.Err()
	}
	defer c.Close()

	c.Send("MULTI")
	c.Send("SREM", tagsKey, tagName)
	c.Send("DEL", tagSubscriptionKey(tagName))
	c.Send("DEL", tagTriggersKey(tagName))
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetTagTriggerIDs gets all triggersIDs by given tagName
func (connector *DbConnector) GetTagTriggerIDs(tagName string) ([]string, error) {
	c := connector.pool.Get()
	if c.Err() != nil {
		return nil, c.Err()
	}
	if c.Err() != nil {
		return nil, c.Err()
	}
	defer c.Close()

	triggerIDs, err := redis.Strings(c.Do("SMEMBERS", tagTriggersKey(tagName)))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("Failed to retrieve tag triggers:%s, err: %s", tagName, err.Error())
	}
	return triggerIDs, nil
}

var tagsKey = "moira-tags"

func tagTriggersKey(tagName string) string {
	return fmt.Sprintf("moira-tag-triggers:%s", tagName)
}

func tagSubscriptionKey(tagName string) string {
	return fmt.Sprintf("moira-tag-subscriptions:%s", tagName)
}
