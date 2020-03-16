package redis

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetAllTriggerIDs gets all moira triggerIDs
func (connector *DbConnector) GetAllTriggerIDs() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIds, err := redis.Strings(c.Do("SMEMBERS", triggersListKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get all triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetLocalTriggerIDs gets moira local triggerIDs
func (connector *DbConnector) GetLocalTriggerIDs() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIds, err := redis.Strings(c.Do("SDIFF", triggersListKey, remoteTriggersListKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetRemoteTriggerIDs gets moira remote triggerIDs
func (connector *DbConnector) GetRemoteTriggerIDs() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerIds, err := redis.Strings(c.Do("SMEMBERS", remoteTriggersListKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get remote triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetTrigger gets trigger and trigger tags by given ID and return it in merged object
func (connector *DbConnector) GetTrigger(triggerID string) (moira.Trigger, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("GET", triggerKey(triggerID))
	c.Send("SMEMBERS", triggerTagsKey(triggerID))
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return moira.Trigger{}, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return connector.getTriggerWithTags(rawResponse[0], rawResponse[1], triggerID)
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
		return nil, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	triggers := make([]*moira.Trigger, len(triggerIDs))
	for i := 0; i < len(rawResponse); i += 2 {
		triggerID := triggerIDs[i/2]
		trigger, err := connector.getTriggerWithTags(rawResponse[i], rawResponse[i+1], triggerID)
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, err
		}
		triggers[i/2] = &trigger
	}
	return triggers, nil
}

// GetPatternTriggerIDs gets trigger list by given pattern
func (connector *DbConnector) GetPatternTriggerIDs(pattern string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIds, err := redis.Strings(c.Do("SMEMBERS", patternTriggersKey(pattern)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pattern triggers for pattern: %s, error: %s", pattern, err.Error())
	}
	return triggerIds, nil
}

// RemovePatternTriggerIDs removes all triggerIDs list accepted to given pattern
func (connector *DbConnector) RemovePatternTriggerIDs(pattern string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", patternTriggersKey(pattern))
	if err != nil {
		return fmt.Errorf("failed delete pattern-triggers: %s, error: %s", pattern, err)
	}
	return nil
}

// SaveTrigger sets trigger data by given trigger and triggerID
// If trigger already exists, then merge old and new trigger patterns and tags list
// and cleanup not used tags and patterns from lists
// If given trigger contains new tags then create it.
// If given trigger has no subscription on it, add it to triggers-without-subscriptions
func (connector *DbConnector) SaveTrigger(triggerID string, trigger *moira.Trigger) error {
	if trigger.IsRemote {
		trigger.Patterns = make([]string, 0)
	}

	var oldTrigger *moira.Trigger
	if existing, err := connector.GetTrigger(triggerID); err == nil {
		oldTrigger = &existing
	} else if err != database.ErrNil {
		return fmt.Errorf("failed to get trigger: %s", err.Error())
	}

	err := connector.updateTrigger(triggerID, trigger, oldTrigger)
	if err != nil {
		return fmt.Errorf("failed to update trigger: %s", err.Error())
	}

	hasSubscriptions, err := connector.triggerHasSubscriptions(trigger)
	if err != nil {
		return fmt.Errorf("failed to check trigger subscriptions: %s", err.Error())
	}

	if !hasSubscriptions {
		err = connector.MarkTriggersAsUnused(triggerID)
	} else {
		err = connector.MarkTriggersAsUsed(triggerID)
	}
	if err != nil {
		return fmt.Errorf("failed to mark trigger as (un)used: %s", err.Error())
	}

	if oldTrigger != nil {
		return connector.cleanupPatternsOutOfUse(moira.GetStringListsDiff(oldTrigger.Patterns, trigger.Patterns))
	}

	return nil
}

func (connector *DbConnector) updateTrigger(triggerID string, newTrigger *moira.Trigger, oldTrigger *moira.Trigger) error {
	bytes, err := reply.GetTriggerBytes(triggerID, newTrigger)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	if oldTrigger != nil {
		for _, pattern := range moira.GetStringListsDiff(oldTrigger.Patterns, newTrigger.Patterns) {
			c.Send("SREM", patternTriggersKey(pattern), triggerID)
		}
		if oldTrigger.IsRemote && !newTrigger.IsRemote {
			c.Send("SREM", remoteTriggersListKey, triggerID)
		}

		for _, tag := range moira.GetStringListsDiff(oldTrigger.Tags, newTrigger.Tags) {
			c.Send("SREM", triggerTagsKey(triggerID), tag)
			c.Send("SREM", tagTriggersKey(tag), triggerID)
		}
	}
	c.Send("SET", triggerKey(triggerID), bytes)
	c.Send("SADD", triggersListKey, triggerID)
	if newTrigger.IsRemote {
		c.Send("SADD", remoteTriggersListKey, triggerID)
	} else {
		for _, pattern := range newTrigger.Patterns {
			c.Send("SADD", patternsListKey, pattern)
			c.Send("SADD", patternTriggersKey(pattern), triggerID)
		}
	}
	for _, tag := range newTrigger.Tags {
		c.Send("SADD", triggerTagsKey(triggerID), tag)
		c.Send("SADD", tagTriggersKey(tag), triggerID)
		c.Send("SADD", tagsKey, tag)
	}
	if connector.source != Cli {
		c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)
	}
	if _, err = c.Do("EXEC"); err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// RemoveTrigger deletes trigger data by given triggerID, delete trigger tag list,
// Deletes triggerID from containing tags triggers list and from containing patterns triggers list
// If containing patterns doesn't used in another triggers, then delete this patterns with metrics data
func (connector *DbConnector) RemoveTrigger(triggerID string) error {
	trigger, err := connector.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil
		}
		return err
	}

	if err = connector.removeTrigger(triggerID, &trigger); err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return connector.cleanupPatternsOutOfUse(trigger.Patterns)
}

func (connector *DbConnector) removeTrigger(triggerID string, trigger *moira.Trigger) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", triggerKey(triggerID))
	c.Send("DEL", triggerTagsKey(triggerID))
	c.Send("DEL", triggerEventsKey(triggerID))
	c.Send("SREM", triggersListKey, triggerID)
	c.Send("SREM", remoteTriggersListKey, triggerID)
	c.Send("SREM", unusedTriggersKey, triggerID)
	for _, tag := range trigger.Tags {
		c.Send("SREM", tagTriggersKey(tag), triggerID)
	}
	for _, pattern := range trigger.Patterns {
		c.Send("SREM", patternTriggersKey(pattern), triggerID)
	}
	c.Send("ZADD", triggersToReindexKey, time.Now().Unix(), triggerID)

	if _, err := c.Do("EXEC"); err != nil {
		return fmt.Errorf("failed to remove trigger %s", err.Error())
	}
	return nil
}

// GetTriggerChecks gets triggers data with tags, lastCheck data and throttling by given triggersIDs
// Len of triggerIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTriggerChecks(triggerIDs []string) ([]*moira.TriggerCheck, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, triggerID := range triggerIDs {
		c.Send("GET", triggerKey(triggerID))
		c.Send("SMEMBERS", triggerTagsKey(triggerID))
		c.Send("GET", metricLastCheckKey(triggerID))
		c.Send("GET", notifierNextKey(triggerID))
	}
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}
	var slices [][]interface{}
	for i := 0; i < len(rawResponse); i += 4 {
		arr := make([]interface{}, 0, 5)
		arr = append(arr, triggerIDs[i/4])
		arr = append(arr, rawResponse[i:i+4]...)
		slices = append(slices, arr)
	}
	triggerChecks := make([]*moira.TriggerCheck, len(slices))
	for i, slice := range slices {
		triggerID := slice[0].(string)
		trigger, err := connector.getTriggerWithTags(slice[1], slice[2], triggerID)
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, err
		}
		lastCheck, err := reply.Check(slice[3], nil)
		if err != nil && err != database.ErrNil {
			return nil, err
		}
		throttling, _ := redis.Int64(slice[4], nil)
		if time.Now().Unix() >= throttling {
			throttling = 0
		}
		triggerChecks[i] = &moira.TriggerCheck{
			Trigger:    trigger,
			LastCheck:  lastCheck,
			Throttling: throttling,
		}
	}
	return triggerChecks, nil
}

func (connector *DbConnector) getTriggerWithTags(triggerRaw interface{}, tagsRaw interface{}, triggerID string) (moira.Trigger, error) {
	trigger, err := reply.Trigger(triggerRaw, nil)
	if err != nil {
		return trigger, err
	}
	triggerTags, err := redis.Strings(tagsRaw, nil)
	if err != nil {
		connector.logger.Errorf("Error getting trigger tags, id: %s, error: %s", triggerID, err.Error())
	}
	if len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	trigger.ID = triggerID
	return trigger, nil
}

func (connector *DbConnector) cleanupPatternsOutOfUse(patterns []string) error {
	for _, pattern := range patterns {
		triggerIDs, err := connector.GetPatternTriggerIDs(pattern)
		if err != nil {
			return err
		}
		if len(triggerIDs) == 0 {
			if err := connector.RemovePatternWithMetrics(pattern); err != nil {
				return err
			}
		}
	}
	return nil
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

var triggersListKey = "moira-triggers-list"
var remoteTriggersListKey = "moira-remote-triggers-list"

func triggerKey(triggerID string) string {
	return "moira-trigger:" + triggerID
}

func triggerTagsKey(triggerID string) string {
	return "moira-trigger-tags:" + triggerID
}

func patternTriggersKey(pattern string) string {
	return "moira-pattern-triggers:" + pattern
}
