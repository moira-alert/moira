package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

//GetTriggerIDs gets all moira triggerIDs
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

var triggersListKey = "moira-triggers-list"

func triggerKey(triggerID string) string {
	return fmt.Sprintf("moira-trigger:%s", triggerID)
}

func triggerTagsKey(triggerID string) string {
	return fmt.Sprintf("moira-trigger-tags:%s", triggerID)
}
