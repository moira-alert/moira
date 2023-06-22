package redis

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetAllTriggerIDs gets all moira triggerIDs
func (connector *DbConnector) GetAllTriggerIDs() ([]string, error) {
	c := *connector.client
	triggerIds, err := c.SMembers(connector.context, triggersListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetLocalTriggerIDs gets moira local triggerIDs
func (connector *DbConnector) GetLocalTriggerIDs() ([]string, error) {
	c := *connector.client
	triggerIds, err := c.SDiff(connector.context, triggersListKey, remoteTriggersListKey, vmselectTriggersListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetRemoteTriggerIDs gets moira remote triggerIDs
func (connector *DbConnector) GetRemoteTriggerIDs() ([]string, error) {
	c := *connector.client
	triggerIds, err := c.SMembers(connector.context, remoteTriggersListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

func (connector *DbConnector) GetVMSelectTriggerIDs() ([]string, error) {
	c := *connector.client
	triggerIds, err := c.SMembers(connector.context, vmselectTriggersListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get vmselect triggers-list: %s", err.Error())
	}
	return triggerIds, nil
}

// GetTrigger gets trigger and trigger tags by given ID and return it in merged object
func (connector *DbConnector) GetTrigger(triggerID string) (moira.Trigger, error) {
	pipe := (*connector.client).TxPipeline()
	trigger := pipe.Get(connector.context, triggerKey(triggerID))
	triggerTags := pipe.SMembers(connector.context, triggerTagsKey(triggerID))
	_, err := pipe.Exec(connector.context)
	if err != nil {
		if err == redis.Nil {
			return moira.Trigger{}, database.ErrNil
		}
		return moira.Trigger{}, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return connector.getTriggerWithTags(trigger, triggerTags, triggerID)
}

// GetTriggers returns triggers data by given ids, len of triggerIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTriggers(triggerIDs []string) ([]*moira.Trigger, error) {
	pipe := (*connector.client).TxPipeline()
	for _, triggerID := range triggerIDs {
		pipe.Get(connector.context, triggerKey(triggerID))
		pipe.SMembers(connector.context, triggerTagsKey(triggerID))
	}
	rawResponse, err := pipe.Exec(connector.context)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	triggers := make([]*moira.Trigger, len(triggerIDs))
	for i := 0; i < len(rawResponse); i += 2 {
		triggerID := triggerIDs[i/2]
		triggerCmd, tagCmd := rawResponse[i].(*redis.StringCmd), rawResponse[i+1].(*redis.StringSliceCmd)
		trigger, err := connector.getTriggerWithTags(triggerCmd, tagCmd, triggerID)
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
	c := *connector.client

	triggerIds, err := c.SMembers(connector.context, patternTriggersKey(pattern)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pattern triggers for pattern: %s, error: %s", pattern, err.Error())
	}
	return triggerIds, nil
}

// RemovePatternTriggerIDs removes all triggerIDs list accepted to given pattern
func (connector *DbConnector) RemovePatternTriggerIDs(pattern string) error {
	c := *connector.client
	_, err := c.Del(connector.context, patternTriggersKey(pattern)).Result()
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
	var oldTrigger *moira.Trigger
	if existing, err := connector.GetTrigger(triggerID); err == nil {
		oldTrigger = &existing
	} else if err != database.ErrNil {
		return fmt.Errorf("failed to get trigger: %s", err.Error())
	}

	connector.preSaveTrigger(trigger, oldTrigger)

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

// GetTriggerIDsStartWith returns triggers which have ID starting with "prefix" parameter.
func (connector *DbConnector) GetTriggerIDsStartWith(prefix string) ([]string, error) {
	triggers, err := connector.GetAllTriggerIDs()
	if err != nil {
		return nil, err
	}

	var matchedTriggers []string
	for _, id := range triggers {
		if strings.HasPrefix(id, prefix) {
			matchedTriggers = append(matchedTriggers, id)
		}
	}

	return matchedTriggers, nil
}

func (connector *DbConnector) updateTrigger(triggerID string, newTrigger *moira.Trigger, oldTrigger *moira.Trigger) error {
	bytes, err := reply.GetTriggerBytes(triggerID, newTrigger)
	if err != nil {
		return err
	}
	pipe := (*connector.client).TxPipeline()
	if oldTrigger != nil {
		for _, pattern := range moira.GetStringListsDiff(oldTrigger.Patterns, newTrigger.Patterns) {
			pipe.SRem(connector.context, patternTriggersKey(pattern), triggerID)
		}

		for _, tag := range moira.GetStringListsDiff(oldTrigger.Tags, newTrigger.Tags) {
			pipe.SRem(connector.context, triggerTagsKey(triggerID), tag)
			pipe.SRem(connector.context, tagTriggersKey(tag), triggerID)
		}

		if newTrigger.TriggerSource != oldTrigger.TriggerSource {
			switch oldTrigger.TriggerSource {
			case moira.GraphiteRemote:
				pipe.SRem(connector.context, remoteTriggersListKey, triggerID)

			case moira.VMSelectRemote:
				pipe.SRem(connector.context, vmselectTriggersListKey, triggerID)
			}
		}
	}
	pipe.Set(connector.context, triggerKey(triggerID), bytes, redis.KeepTTL)
	pipe.SAdd(connector.context, triggersListKey, triggerID)

	switch newTrigger.TriggerSource {
	case moira.GraphiteRemote:
		pipe.SAdd(connector.context, remoteTriggersListKey, triggerID)

	case moira.VMSelectRemote:
		pipe.SAdd(connector.context, vmselectTriggersListKey, triggerID)

	case moira.GraphiteLocal:
		for _, pattern := range newTrigger.Patterns {
			pipe.SAdd(connector.context, patternsListKey, pattern)
			pipe.SAdd(connector.context, patternTriggersKey(pattern), triggerID)
		}
	}

	for _, tag := range newTrigger.Tags {
		pipe.SAdd(connector.context, triggerTagsKey(triggerID), tag)
		pipe.SAdd(connector.context, tagTriggersKey(tag), triggerID)
		pipe.SAdd(connector.context, tagsKey, tag)
	}
	if connector.source != Cli {
		z := &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID}
		pipe.ZAdd(connector.context, triggersToReindexKey, z)
	}
	if _, err = pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) preSaveTrigger(newTrigger *moira.Trigger, oldTrigger *moira.Trigger) {
	if newTrigger.TriggerSource != moira.GraphiteLocal {
		newTrigger.Patterns = make([]string, 0)
	}

	now := connector.clock.Now().Unix()
	newTrigger.UpdatedAt = &now
	if oldTrigger != nil {
		newTrigger.CreatedAt = oldTrigger.CreatedAt
		newTrigger.CreatedBy = oldTrigger.CreatedBy
	} else {
		newTrigger.CreatedAt = &now
		newTrigger.CreatedBy = newTrigger.UpdatedBy
	}
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
	pipe := (*connector.client).TxPipeline()
	pipe.Del(connector.context, triggerKey(triggerID))
	pipe.Del(connector.context, triggerTagsKey(triggerID))
	pipe.Del(connector.context, triggerEventsKey(triggerID))
	pipe.SRem(connector.context, triggersListKey, triggerID)

	switch trigger.TriggerSource {
	case moira.GraphiteRemote:
		pipe.SRem(connector.context, remoteTriggersListKey, triggerID)

	case moira.VMSelectRemote:
		pipe.SRem(connector.context, vmselectTriggersListKey, triggerID)
	}

	pipe.SRem(connector.context, unusedTriggersKey, triggerID)
	for _, tag := range trigger.Tags {
		pipe.SRem(connector.context, tagTriggersKey(tag), triggerID)
	}
	for _, pattern := range trigger.Patterns {
		pipe.SRem(connector.context, patternTriggersKey(pattern), triggerID)
	}
	z := &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID}
	pipe.ZAdd(connector.context, triggersToReindexKey, z)

	pipe = appendRemoveTriggerLastCheckToRedisPipeline(connector.context, pipe, triggerID)

	if _, err := pipe.Exec(connector.context); err != nil {
		return fmt.Errorf("failed to remove trigger %s", err.Error())
	}
	return nil
}

// GetTriggerChecks gets triggers data with tags, lastCheck data and throttling by given triggersIDs
// Len of triggerIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetTriggerChecks(triggerIDs []string) ([]*moira.TriggerCheck, error) {
	pipe := (*connector.client).TxPipeline()
	for _, triggerID := range triggerIDs {
		pipe.Get(connector.context, triggerKey(triggerID))
		pipe.SMembers(connector.context, triggerTagsKey(triggerID))
		pipe.Get(connector.context, metricLastCheckKey(triggerID))
		pipe.Get(connector.context, notifierNextKey(triggerID))
	}
	rawResponse, err := pipe.Exec(connector.context)

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}
	var slices [][]interface{}
	for i := 0; i < len(rawResponse); i += 4 {
		arr := make([]interface{}, 0, 5)
		arr = append(arr, triggerIDs[i/4])
		arr = append(arr, rawResponse[i], rawResponse[i+1], rawResponse[i+2], rawResponse[i+3])
		slices = append(slices, arr)
	}
	triggerChecks := make([]*moira.TriggerCheck, len(slices))
	for i, slice := range slices {
		triggerID := slice[0].(string)
		triggerCmd, tagCmd := slice[1].(*redis.StringCmd), slice[2].(*redis.StringSliceCmd)
		trigger, err := connector.getTriggerWithTags(triggerCmd, tagCmd, triggerID)
		if err != nil {
			if err == database.ErrNil {
				continue
			}
			return nil, err
		}
		lastCheck, err := reply.Check(slice[3].(*redis.StringCmd))
		if err != nil && err != database.ErrNil {
			return nil, err
		}
		throttlingStr, _ := slice[4].(*redis.StringCmd).Result()
		throttling, _ := strconv.ParseInt(throttlingStr, 10, 64)
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

func (connector *DbConnector) getTriggerWithTags(triggerRaw *redis.StringCmd, tagsRaw *redis.StringSliceCmd, triggerID string) (moira.Trigger, error) {
	trigger, err := reply.Trigger(triggerRaw)
	if err != nil {
		return trigger, err
	}
	triggerTags, err := tagsRaw.Result()
	if err != nil {
		connector.logger.Error().
			String(moira.LogFieldNameTriggerID, triggerID).
			Error(err).
			Msg("Error getting trigger tags")
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

var triggersListKey = "{moira-triggers-list}:moira-triggers-list"
var remoteTriggersListKey = "{moira-triggers-list}:moira-remote-triggers-list"
var vmselectTriggersListKey = "{moira-triggers-list}:moira-vmselect-triggers-list"

func triggerKey(triggerID string) string {
	return "moira-trigger:" + triggerID
}

func triggerTagsKey(triggerID string) string {
	return "moira-trigger-tags:" + triggerID
}

func patternTriggersKey(pattern string) string {
	return "moira-pattern-triggers:" + pattern
}
