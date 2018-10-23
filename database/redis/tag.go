package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira/database"
)

// GetTagNames returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := connector.pool.Get()
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

func (connector *DbConnector) getTagsTriggerIDs(tagNames []string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, tagName := range tagNames {
		c.Send("SMEMBERS", tagTriggersKey(tagName))
	}

	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	triggerIDsByTags := make(map[string]bool, 0)

	for _, triggersArray := range rawResponse {
		tagTriggerIDs, err := redis.Strings(triggersArray, nil)
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, fmt.Errorf("failed to retrieve tags triggers: %s", err.Error())
		}
		for _, triggerID := range tagTriggerIDs {
			triggerIDsByTags[triggerID] = true
		}
	}

	triggerIDs := make([]string, 0)
	for triggerID := range triggerIDsByTags {
		triggerIDs = append(triggerIDs, triggerID)
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
