package redis

import (
	"fmt"

	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"strconv"
	"time"
)

// GetUserContacts - Returns contacts ids by given login from set {0}
func (connector *DbConnector) GetUserContacts(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var subscriptions []string

	values, err := redis.Values(c.Do("SMEMBERS", fmt.Sprintf("moira-user-contacts:%s", login)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	if err := redis.ScanSlice(values, &subscriptions); err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	return subscriptions, nil
}

//GetUserSubscriptions - Returns subscriptions ids by given login from set {0}
func (connector *DbConnector) GetUserSubscriptions(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var subscriptions []string

	values, err := redis.Values(c.Do("SMEMBERS", fmt.Sprintf("moira-user-subscriptions:%s", login)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	if err := redis.ScanSlice(values, &subscriptions); err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for user login %s: %s", login, err.Error())
	}
	return subscriptions, nil
}

//GetTags returns all tags from set with tag data
func (connector *DbConnector) GetTagNames() ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var tagNames []string

	values, err := redis.Values(c.Do("SMEMBERS", "moira-tags"))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve moira-tags: %s", err.Error())
	}
	if err := redis.ScanSlice(values, &tagNames); err != nil {
		return nil, fmt.Errorf("Failed to retrieve moira-tags: %s", err.Error())
	}
	return tagNames, nil
}

//GetTag returns tag data by key
func (connector *DbConnector) GetTag(tagName string) (moira.TagData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var tag moira.TagData

	tagString, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-tag:%s", tagName)))
	if err != nil {
		if err == redis.ErrNil {
			return tag, nil
		}
		return tag, fmt.Errorf("Failed to get tag data for id %s: %s", tagName, err.Error())
	}
	if err := json.Unmarshal(tagString, &tag); err != nil {
		return tag, fmt.Errorf("Failed to parse tag json %s: %s", tagString, err.Error())
	}

	return tag, nil
}

func (connector *DbConnector) GetFilteredTriggersIds(tagNames []string, onlyErrors bool) ([]string, int64, error) {
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
		for _, triggerId := range triggerIds {
			triggerIdsMap[triggerId] = true
		}

		triggerIdsByTags = append(triggerIdsByTags, triggerIdsMap)
	}

	total := make([]string, 0)
	for _, triggerId := range triggerIdsChecks {
		valid := true
		for _, triggerIdsByTag := range triggerIdsByTags {
			if _, ok := triggerIdsByTag[triggerId]; !ok {
				valid = false
				break
			}
		}
		if valid {
			total = append(total, triggerId)
		}
	}
	return total, int64(len(total)), nil
}

func (connector *DbConnector) GetTriggerIds() ([]string, int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZREVRANGE", "moira-triggers-checks", 0, -1)
	c.Send("ZCARD", "moira-triggers-checks")
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, err
	}
	triggerIds, err := redis.Strings(rawResponse[0], nil)
	if err != nil {
		return nil, 0, err
	}
	total, err := redis.Int(rawResponse[1], nil)
	if err != nil {
		return nil, 0, err
	}
	return triggerIds, int64(total), nil
}

func (connector *DbConnector) GetTriggersChecks(triggerIds []string) ([]moira.TriggerChecks, error) {
	c := connector.pool.Get()
	defer c.Close()
	var triggerChecks []moira.TriggerChecks

	c.Send("MULTI")
	for _, triggerCheckId := range triggerIds {
		c.Send("GET", fmt.Sprintf("moira-trigger:%s", triggerCheckId))
		c.Send("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerCheckId))
		c.Send("GET", fmt.Sprintf("moira-metric-last-check:%s", triggerCheckId))
		c.Send("GET", fmt.Sprintf("moira-notifier-next:%s", triggerCheckId))
	}
	rawResponce, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, err
	}

	var slices [][]interface{}
	for i := 0; i < len(rawResponce); i += 4 {
		arr := make([]interface{}, 0, 5)
		arr = append(arr, triggerIds[i/4])
		arr = append(arr, rawResponce[i:i+4]...)
		slices = append(slices, arr)
	}
	for _, slice := range slices {
		triggerId := slice[0]
		var trigger = &TriggerChecksDataStorageElement{}

		triggerBytes, err := redis.Bytes(slice[1], nil)
		if err != nil {
			connector.logger.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerId, err.Error())
			continue
		}
		if err := json.Unmarshal(triggerBytes, &trigger); err != nil {
			connector.logger.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
			continue
		}
		if trigger == nil {
			continue
		}
		triggerTags, err := redis.Strings(slice[2], nil)
		if err != nil {
			connector.logger.Errorf("Error getting trigger-tags, id: %s, error: %s", triggerId, err.Error())
		}

		lastCheckBytes, err := redis.Bytes(slice[3], nil)
		if err != nil {
			connector.logger.Errorf("Error getting metric-last-check, id: %s, error: %s", triggerId, err.Error())
		}

		var lastCheck = moira.CheckData{}
		err = json.Unmarshal(lastCheckBytes, &lastCheck)
		if err != nil {
			connector.logger.Errorf("Failed to parse lastCheck json %s: %s", lastCheckBytes, err.Error())
		}

		throttling, err := redis.Int64(slice[4], nil)
		if err != nil {
			connector.logger.Errorf("Error getting moira-notifier-next, id: %s, error: %s", triggerId, err.Error())
		}

		trigger.ID = triggerId.(string)
		trigger.LastCheck = lastCheck
		if throttling > time.Now().Unix() {
			trigger.Throttling = throttling
		}
		if triggerTags != nil && len(triggerTags) > 0 {
			trigger.Tags = triggerTags
		}

		triggerChecks = append(triggerChecks, *toTriggerCheckData(trigger))
	}

	return triggerChecks, nil
}

func (connector *DbConnector) GetTags(tagNames []string) (map[string]moira.TagData, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, tagName := range tagNames {
		c.Send("GET", fmt.Sprintf("moira-tag:%s", tagName))
	}
	rawResponse, err := redis.ByteSlices(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	allTags := make(map[string]moira.TagData)
	for i, tagBytes := range rawResponse {
		var tag moira.TagData
		if err := json.Unmarshal(tagBytes, &tag); err != nil {
			connector.logger.Warningf("Failed to parse tag json %s: %s", tagBytes, err.Error())
			allTags[tagNames[i]] = moira.TagData{}
			continue
		}
		allTags[tagNames[i]] = tag
	}

	return allTags, nil
}

func (connector *DbConnector) GetTrigger(triggerId string) (*moira.Trigger, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("GET", fmt.Sprintf("moira-trigger:%s", triggerId))
	c.Send("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerId))
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	var trigger = &TriggerChecksDataStorageElement{}
	triggerBytes, err := redis.Bytes(rawResponse[0], nil)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerId, err.Error())
	}
	if err := json.Unmarshal(triggerBytes, &trigger); err != nil {
		return nil, fmt.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
	}

	triggerTags, err := redis.Strings(rawResponse[1], nil)
	if err != nil {
		connector.logger.Errorf("Error getting trigger-tags, id: %s, error: %s", triggerId, err.Error())
	}

	if triggerTags != nil && len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	return toTrigger(trigger), nil
}

func (connector *DbConnector) GetTriggerLastCheck(triggerId string) (*moira.CheckData, error) {
	c := connector.pool.Get()
	defer c.Close()

	lastCheckBytes, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-metric-last-check:%s", triggerId)))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}

		return nil, fmt.Errorf("Error getting metric-last-check, id: %s, error: %s", triggerId, err.Error())
	}

	var lastCheck = moira.CheckData{}
	err = json.Unmarshal(lastCheckBytes, &lastCheck)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse lastCheck json %s: %s", lastCheckBytes, err.Error())
	}

	return &lastCheck, nil
}

func (connector *DbConnector) GetEvents(triggerId string, start int64, size int64) ([]*moira.EventData, error) {
	c := connector.pool.Get()
	defer c.Close()

	eventsDataString, err := redis.Strings(c.Do("ZREVRANGE", fmt.Sprintf("moira-trigger-events:%s", triggerId), start, start+size))
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.EventData, 0), nil
		}
		return nil, fmt.Errorf("Failed to get range for moira-trigger-events, triggerId: %s, error: %s", triggerId, err.Error())
	}

	eventDatas := make([]*moira.EventData, 0, len(eventsDataString))

	for _, eventDataString := range eventsDataString {
		eventData := &moira.EventData{}
		if err := json.Unmarshal([]byte(eventDataString), eventData); err != nil {
			connector.logger.Warningf("Failed to parse scheduled json notification %s: %s", eventDataString, err.Error())
			continue
		}
		eventDatas = append(eventDatas, eventData)
	}

	return eventDatas, nil
}

type TriggerChecksDataStorageElement struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Desc            string             `json:"desc"`
	Targets         []string           `json:"targets"`
	WarnValue       float64            `json:"warn_value"`
	ErrorValue      float64            `json:"error_value"`
	Tags            []string           `json:"tags"`
	TtlState        string             `json:"ttl_state"`
	Schedule        moira.ScheduleData `json:"sched"`
	Expression      string             `json:"expression"`
	Patterns        []string           `json:"patterns"`
	IsSimpleTrigger bool               `json:"is_simple_trigger"`
	Ttl             string             `json:"ttl"`
	Throttling      int64              `json:"throttling"`
	LastCheck       moira.CheckData    `json:"last_check"`
}

func toTriggerCheckData(storageElement *TriggerChecksDataStorageElement) *moira.TriggerChecks {
	return &moira.TriggerChecks{
		Trigger:    *toTrigger(storageElement),
		Throttling: storageElement.Throttling,
		LastCheck:  storageElement.LastCheck,
	}
}

func toTrigger(storageElement *TriggerChecksDataStorageElement) *moira.Trigger {
	ttl, _ := strconv.ParseInt(storageElement.Ttl, 10, 64)
	return &moira.Trigger{
		ID:              storageElement.ID,
		Name:            storageElement.Name,
		Desc:            storageElement.Desc,
		Targets:         storageElement.Targets,
		WarnValue:       storageElement.WarnValue,
		ErrorValue:      storageElement.ErrorValue,
		Tags:            storageElement.Tags,
		TtlState:        storageElement.TtlState,
		Schedule:        storageElement.Schedule,
		Expression:      storageElement.Expression,
		Patterns:        storageElement.Patterns,
		IsSimpleTrigger: storageElement.IsSimpleTrigger,
		Ttl:             ttl,
	}
}
