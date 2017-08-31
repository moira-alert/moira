package redis

import (
	"fmt"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

//GetTriggerIDs gets all moira triggerIDs, if no value, return database.ErrNil error
func (connector *DbConnector) GetTriggerIDs() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIds, err := redis.Strings(c.Do("SMEMBERS", triggersListKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to get triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

//GetTrigger gets trigger and trigger tags by given ID and return it in merged object
func (connector *DbConnector) GetTrigger(triggerID string) (moira.Trigger, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("GET", triggerKey(triggerID))
	c.Send("SMEMBERS", triggerTagsKey(triggerID))
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return moira.Trigger{}, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	trigger, err := reply.Trigger(rawResponse[0], nil)
	if err != nil {
		return trigger, err
	}
	triggerTags, err := redis.Strings(rawResponse[1], nil)
	if err != nil {
		connector.logger.Errorf("Error getting trigger tags, id: %s, error: %s", triggerID, err.Error())
	}
	trigger.ID = triggerID
	if len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	return trigger, err
}

// GetTriggers returns triggers data by given ids, len of triggerIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTriggers(triggerIDs []string) ([]*moira.Trigger, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("GET", triggerKey(triggerID))
		c.Send("SMEMBERS", triggerTagsKey(triggerID))
	}
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	triggers := make([]*moira.Trigger, len(triggerIDs))
	for i := 0; i < len(rawResponse); i += 2 {
		triggerID := triggerIDs[i/2]
		trigger, err := reply.Trigger(rawResponse[i], nil)
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, err
		}
		triggerTags, err := redis.Strings(rawResponse[i+1], nil)
		if err != nil {
			connector.logger.Errorf("Error getting trigger tags, id: %s, error: %s", triggerID, err.Error())
		}
		trigger.ID = triggerID
		if len(triggerTags) > 0 {
			trigger.Tags = triggerTags
		}
		triggers = append(triggers, &trigger)
	}
	return triggers, nil
}

//GetPatternTriggerIDs gets trigger list by given pattern
func (connector *DbConnector) GetPatternTriggerIDs(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIds, err := redis.Strings(c.Do("SMEMBERS", patternTriggersKey(pattern)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve pattern triggers for pattern: %s, error: %s", pattern, err.Error())
	}
	return triggerIds, nil
}

//RemovePatternTriggerIDs removes all triggerIDs list accepted to given pattern
func (connector *DbConnector) RemovePatternTriggerIDs(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", patternTriggersKey(pattern))
	if err != nil {
		return fmt.Errorf("Failed delete pattern-triggers: %s, error: %s", pattern, err)
	}
	return nil
}

//SaveTrigger sets trigger data by given trigger and triggerID
//If trigger already exists, then merge old and new trigger patterns and tags list
//and cleanup not used tags and patterns from lists
//If given trigger contains new tags then create it
func (connector *DbConnector) SaveTrigger(triggerID string, trigger *moira.Trigger) error {
	existing, errGetTrigger := connector.GetTrigger(triggerID)
	if errGetTrigger != nil && errGetTrigger != database.ErrNil {
		return errGetTrigger
	}
	bytes, err := reply.GetTriggerBytes(triggerID, trigger)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	cleanupPatterns := make([]string, 0)
	if errGetTrigger != database.ErrNil {
		for _, pattern := range leftJoin(existing.Patterns, trigger.Patterns) {
			c.Send("SREM", patternTriggersKey(pattern), triggerID)
			cleanupPatterns = append(cleanupPatterns, pattern)
		}
		for _, tag := range leftJoin(existing.Tags, trigger.Tags) {
			c.Send("SREM", triggerTagsKey(triggerID), tag)
			c.Send("SREM", moiraTagTriggers(tag), triggerID)
		}
	}
	c.Do("SET", triggerKey(triggerID), bytes)
	c.Do("SADD", triggersListKey, triggerID)
	for _, pattern := range trigger.Patterns {
		c.Do("SADD", moiraPatternsList, pattern)
		c.Do("SADD", patternTriggersKey(pattern), triggerID)
	}
	for _, tag := range trigger.Tags {
		c.Send("SADD", triggerTagsKey(triggerID), tag)
		c.Send("SADD", moiraTagTriggers(tag), triggerID)
		c.Send("SADD", moiraTags, tag)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	for _, pattern := range cleanupPatterns {
		triggerIDs, err := connector.GetPatternTriggerIDs(pattern)
		if err != nil {
			return err
		}
		if len(triggerIDs) == 0 {
			connector.RemovePatternTriggerIDs(pattern)
			connector.RemovePattern(pattern)
			connector.RemovePatternsMetrics([]string{pattern})
		}
	}
	return nil
}

//RemoveTrigger deletes trigger data by given triggerID, delete trigger tag list,
//Deletes triggerID from containing tags triggers list and from containing patterns triggers list
//If containing patterns doesn't used in another triggers, then delete this patterns with metrics data
func (connector *DbConnector) RemoveTrigger(triggerID string) error {
	trigger, err := connector.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil
		}
		return err
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", triggerKey(triggerID))
	c.Send("DEL", triggerTagsKey(triggerID))
	c.Send("SREM", triggersListKey, triggerID)
	for _, tag := range trigger.Tags {
		c.Send("SREM", moiraTagTriggers(tag), triggerID)
	}
	for _, pattern := range trigger.Patterns {
		c.Send("SREM", patternTriggersKey(pattern), triggerID)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	for _, pattern := range trigger.Patterns {
		count, err := redis.Int64(c.Do("SCARD", patternTriggersKey(pattern)))
		if err != nil {
			return fmt.Errorf("Failed to SCARD pattern triggers: %s", err.Error())
		}
		if count == 0 {
			if err := connector.RemovePatternWithMetrics(pattern); err != nil {
				return err
			}
		}
	}
	return nil
}

var triggersListKey = "moira-triggers-list"

func triggerKey(triggerID string) string {
	return fmt.Sprintf("moira-trigger:%s", triggerID)
}

func triggerTagsKey(triggerID string) string {
	return fmt.Sprintf("moira-trigger-tags:%s", triggerID)
}

func patternTriggersKey(pattern string) string {
	return fmt.Sprintf("moira-pattern-triggers:%s", pattern)
}
