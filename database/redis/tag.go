package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	tagNames, err := redis.Strings(c.Do("SMEMBERS", moiraTags))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags: %s", err.Error())
	}
	return tagNames, nil
}

// RemoveTag deletes tag from tags list, deletes triggerIDs and subscriptionsIDs lists by given tag
func (connector *DbConnector) RemoveTag(tagName string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SREM", moiraTags, tagName)
	c.Send("DEL", moiraTagSubscription(tagName))
	c.Send("DEL", moiraTagTriggers(tagName))
	c.Send("DEL", moiraTag(tagName))
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetTagTriggerIDs gets all triggersIDs by given tagName
func (connector *DbConnector) GetTagTriggerIDs(tagName string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIDs, err := redis.Strings(c.Do("SMEMBERS", moiraTagTriggers(tagName)))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("Failed to retrieve tag triggers:%s, err: %s", tagName, err.Error())
	}
	return triggerIDs, nil
}

var moiraTags = "moira-tags"

func moiraTag(tagName string) string {
	return fmt.Sprintf("moira-tag:%s", tagName)
}

func moiraTagTriggers(tagName string) string {
	return fmt.Sprintf("moira-tag-triggers:%s", tagName)
}

func moiraTagSubscription(tagName string) string {
	return fmt.Sprintf("moira-tag-subscriptions:%s", tagName)
}
