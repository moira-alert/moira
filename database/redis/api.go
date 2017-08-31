package redis

import (
	"fmt"

	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"time"
)

func (connector *DbConnector) GetTriggerChecks(triggerCheckIDs []string) ([]moira.TriggerChecks, error) {
	c := connector.pool.Get()
	defer c.Close()
	triggerChecks := make([]moira.TriggerChecks, 0)

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
