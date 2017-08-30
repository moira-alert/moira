package redis

import (
	"fmt"

	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"time"
)

func (connector *DbConnector) GetFilteredTriggerCheckIds(tagNames []string, onlyErrors bool) ([]string, int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZREVRANGE", "moira-triggers-checks", 0, -1)
	commandsArray := make([]string, 0)
	for _, tagName := range tagNames {
		commandsArray = append(commandsArray, fmt.Sprintf("moira-tag-triggers:%s", tagName))
	}
	if onlyErrors {
		commandsArray = append(commandsArray, "moira-bad-state-triggers")
	}
	for _, command := range commandsArray {
		c.Send("SMEMBERS", command)
	}
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, err
	}

	triggerIdsByTags := make([]map[string]bool, 0)
	var triggerIdsChecks []string

	values, err := redis.Values(rawResponse[0], nil)
	if err != nil {
		return nil, 0, err
	}
	if err := redis.ScanSlice(values, &triggerIdsChecks); err != nil {
		return nil, 0, fmt.Errorf("Failed to retrieve moira-triggers-checks: %s", err.Error())
	}
	for _, triggersArray := range rawResponse[1:] {
		var triggerIds []string
		values, err := redis.Values(triggersArray, nil)
		if err != nil {
			connector.logger.Error(err.Error())
			continue
		}
		if err := redis.ScanSlice(values, &triggerIds); err != nil {
			connector.logger.Errorf("Failed to retrieve moira-tags-triggers: %s", err.Error())
			continue
		}

		triggerIdsMap := make(map[string]bool)
		for _, triggerID := range triggerIds {
			triggerIdsMap[triggerID] = true
		}

		triggerIdsByTags = append(triggerIdsByTags, triggerIdsMap)
	}

	total := make([]string, 0)
	for _, triggerID := range triggerIdsChecks {
		valid := true
		for _, triggerIdsByTag := range triggerIdsByTags {
			if _, ok := triggerIdsByTag[triggerID]; !ok {
				valid = false
				break
			}
		}
		if valid {
			total = append(total, triggerID)
		}
	}
	return total, int64(len(total)), nil
}

func (connector *DbConnector) GetTriggerChecks(triggerCheckIDs []string) ([]moira.TriggerChecks, error) {
	c := connector.pool.Get()
	defer c.Close()
	var triggerChecks []moira.TriggerChecks

	c.Send("MULTI")
	for _, triggerCheckID := range triggerCheckIDs {
		c.Send("GET", fmt.Sprintf("moira-trigger:%s", triggerCheckID))
		c.Send("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerCheckID))
		c.Send("GET", fmt.Sprintf("moira-metric-last-check:%s", triggerCheckID))
		c.Send("GET", fmt.Sprintf("moira-notifier-next:%s", triggerCheckID))
	}
	rawResponce, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, err
	}

	var slices [][]interface{}
	for i := 0; i < len(rawResponce); i += 4 {
		arr := make([]interface{}, 0, 5)
		arr = append(arr, triggerCheckIDs[i/4])
		arr = append(arr, rawResponce[i:i+4]...)
		slices = append(slices, arr)
	}
	for _, slice := range slices {
		triggerID := slice[0].(string)
		var triggerSE = &triggerStorageElement{}

		triggerBytes, err := redis.Bytes(slice[1], nil)
		if err != nil {
			if err != redis.ErrNil {
				connector.logger.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerID, err.Error())
			}
			continue
		}
		if err = json.Unmarshal(triggerBytes, &triggerSE); err != nil {
			connector.logger.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
			continue
		}
		if triggerSE == nil {
			continue
		}
		triggerTags, err := redis.Strings(slice[2], nil)
		if err != nil {
			if err != redis.ErrNil {
				connector.logger.Errorf("Error getting trigger-tags, id: %s, error: %s", triggerID, err.Error())
			}
		}

		lastCheckBytes, err := redis.Bytes(slice[3], nil)
		if err != nil {
			connector.logger.Errorf("Error getting metric-last-check, id: %s, error: %s", triggerID, err.Error())
		}

		var lastCheck = moira.CheckData{}
		err = json.Unmarshal(lastCheckBytes, &lastCheck)
		if err != nil {
			connector.logger.Errorf("Failed to parse lastCheck json %s: %s", lastCheckBytes, err.Error())
		}

		throttling, err := redis.Int64(slice[4], nil)
		if err != nil {
			if err != redis.ErrNil {
				connector.logger.Errorf("Error getting moira-notifier-next, id: %s, error: %s", triggerID, err.Error())
			}
		}

		triggerCheck := moira.TriggerChecks{
			Trigger: *toTrigger(triggerSE, triggerID),
		}

		triggerCheck.LastCheck = lastCheck
		if throttling > time.Now().Unix() {
			triggerCheck.Throttling = throttling
		}
		if len(triggerTags) > 0 {
			triggerCheck.Tags = triggerTags
		}

		triggerChecks = append(triggerChecks, triggerCheck)
	}

	return triggerChecks, nil
}

func (connector *DbConnector) convertTriggerWithTags(triggerInterface interface{}, triggerTagsInterface interface{}, triggerID string) (*triggerStorageElement, error) {
	trigger := &triggerStorageElement{}
	triggerBytes, err := redis.Bytes(triggerInterface, nil)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerID, err.Error())
	}
	if err = json.Unmarshal(triggerBytes, trigger); err != nil {
		return nil, fmt.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
	}
	triggerTags, err := redis.Strings(triggerTagsInterface, nil)
	if err != nil {
		connector.logger.Errorf("Error getting trigger-tags, id: %s, error: %s", triggerID, err.Error())
	}
	if len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	return trigger, nil
}
