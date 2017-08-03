package redis

import (
	"fmt"

	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"strconv"
	"strings"
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

//GetUserSubscriptionIds - Returns subscriptions ids by given login from set {0}
func (connector *DbConnector) GetUserSubscriptionIds(login string) ([]string, error) {
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

func (connector *DbConnector) GetTriggerCheckIds() ([]string, int64, error) {
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

func (connector *DbConnector) GetTriggerChecks(triggerCheckIds []string) ([]moira.TriggerChecks, error) {
	c := connector.pool.Get()
	defer c.Close()
	var triggerChecks []moira.TriggerChecks

	c.Send("MULTI")
	for _, triggerCheckId := range triggerCheckIds {
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
		arr = append(arr, triggerCheckIds[i/4])
		arr = append(arr, rawResponce[i:i+4]...)
		slices = append(slices, arr)
	}
	for _, slice := range slices {
		triggerId := slice[0].(string)
		var triggerSE = &TriggerStorageElement{}

		triggerBytes, err := redis.Bytes(slice[1], nil)
		if err != nil {
			connector.logger.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerId, err.Error())
			continue
		}
		if err := json.Unmarshal(triggerBytes, &triggerSE); err != nil {
			connector.logger.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
			continue
		}
		if triggerSE == nil {
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

		triggerCheck := moira.TriggerChecks{
			Trigger: *toTrigger(triggerSE, triggerId),
		}

		triggerCheck.LastCheck = lastCheck
		if throttling > time.Now().Unix() {
			triggerCheck.Throttling = throttling
		}
		if triggerTags != nil && len(triggerTags) > 0 {
			triggerCheck.Tags = triggerTags
		}

		triggerChecks = append(triggerChecks, triggerCheck)
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
	triggerSE, err := connector.convertTriggerWithTags(rawResponse[0], rawResponse[1], triggerId)
	if err != nil {
		return nil, err
	}
	if triggerSE == nil {
		return nil, nil
	}
	return toTrigger(triggerSE, triggerId), nil
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

func (connector *DbConnector) GetSubscriptions(subscriptionIds []string) ([]moira.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, id := range subscriptionIds {
		c.Send("GET", fmt.Sprintf("moira-subscription:%s", id))
	}
	subscriptionsBytes, err := redis.ByteSlices(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}

	subscriptions := make([]moira.SubscriptionData, 0, len(subscriptionIds))

	for i, bytes := range subscriptionsBytes {
		sub, err := connector.convertSubscription(bytes)
		if err != nil {
			connector.logger.Warningf(err.Error())
			continue
		}
		sub.ID = subscriptionIds[i]
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, nil
}

func (connector *DbConnector) UpdateSubscription(subscription *moira.SubscriptionData) error {
	oldSubscription, err := connector.GetSubscription(subscription.ID)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(subscription)
	if err != nil {
		return err
	}
	subscriptionId := subscription.ID

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, tag := range oldSubscription.Tags {
		c.Send("SREM", fmt.Sprintf("moira-tag-subscriptions:%s", tag), subscriptionId)
	}
	for _, tag := range subscription.Tags {
		c.Send("SADD", fmt.Sprintf("moira-tag-subscriptions:%s", tag), subscriptionId)
	}
	c.Send("SADD", fmt.Sprintf("moira-user-subscriptions:%s", subscription.User), subscriptionId)
	c.Send("SET", fmt.Sprintf("moira-subscription:%s", subscriptionId), bytes)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) CreateSubscription(subscription *moira.SubscriptionData) error {
	bytes, err := json.Marshal(subscription)
	if err != nil {
		return err
	}
	subscriptionId := subscription.ID
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, tag := range subscription.Tags {
		c.Send("SADD", fmt.Sprintf("moira-tag-subscriptions:%s", tag), subscriptionId)
	}
	c.Send("SADD", fmt.Sprintf("moira-user-subscriptions:%s", subscription.User), subscriptionId)
	c.Send("SET", fmt.Sprintf("moira-subscription:%s", subscriptionId), bytes)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) WriteSubscriptions(subscriptions []*moira.SubscriptionData) error {
	subscriptionsBytes := make([][]byte, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		bytes, err := json.Marshal(subscription)
		if err != nil {
			return err
		}
		subscriptionsBytes = append(subscriptionsBytes, bytes)
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for i, bytes := range subscriptionsBytes {
		c.Send("SET", fmt.Sprintf("moira-subscription:%s", subscriptions[i].ID), bytes)
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) DeleteSubscription(subscriptionId string, userLogin string) error {
	subscription, err := connector.GetSubscription(subscriptionId)
	if err != nil {
		return nil
	}
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("SREM", fmt.Sprintf("moira-user-subscriptions:%s", userLogin), subscriptionId)
	for _, tag := range subscription.Tags {
		c.Send("SREM", fmt.Sprintf("moira-tag-subscriptions:%s", tag), subscriptionId)
	}
	c.Send("DEL", fmt.Sprintf("moira-subscription:%s", subscriptionId))
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) DeleteContact(contactId string, userLogin string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", fmt.Sprintf("moira-contact:%s", contactId))
	c.Send("SREM", fmt.Sprintf("moira-user-contacts:%s", userLogin), contactId)
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) WriteContact(contact *moira.ContactData) error {
	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SET", fmt.Sprintf("moira-contact:%s", contact.ID), contactString)
	c.Send("SADD", fmt.Sprintf("moira-user-contacts:%s", contact.User), contact.ID)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) PushEvent(event *moira.EventData, ui bool) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("LPUSH", "moira-trigger-events", eventBytes)
	//todo легально? может указатель правильнее?
	if event.TriggerID != "" {
		c.Send("ZADD", fmt.Sprintf("moira-trigger-events:%s", event.TriggerID), event.Timestamp, eventBytes)
		c.Send("ZREMRANGEBYSCORE", fmt.Sprintf("moira-trigger-events:%s", event.TriggerID), "-inf", time.Now().Unix()-3600*24*30)
	}
	if ui {
		c.Send("LPUSH", "moira-trigger-events-ui", eventBytes)
		c.Send("LTRIM", 0, 100)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) SetTagMaintenance(name string, data moira.TagData) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	_, err = c.Do("SET", fmt.Sprintf("moira-tag:%s", name), bytes)
	if err != nil {
		return fmt.Errorf("Failed to set moira-tag:%s, err: %s", name, err.Error())
	}
	return err
}

func (connector *DbConnector) GetTagTriggerIds(tagName string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	triggerIds, err := redis.Strings(c.Do("SMEMBERS", fmt.Sprintf("moira-tag-triggers:%s", tagName)))
	if err != nil {
		if err == redis.ErrNil {
			return make([]string, 0), nil
		}
		return nil, fmt.Errorf("Failed to moira-tag-triggers:%s, err: %s", tagName, err.Error())
	}
	return triggerIds, nil
}

func (connector *DbConnector) DeleteTag(tagName string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SREM", "moira-tags", tagName)
	c.Send("DEL", fmt.Sprintf("moira-tag-subscriptions:%s", tagName))
	c.Send("DEL", fmt.Sprintf("moira-tag-triggers:%s", tagName))
	c.Send("DEL", fmt.Sprintf("moira-tag:%s", tagName))
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

func (connector *DbConnector) SetTriggerMetricsMaintenance(triggerId string, metrics map[string]int64) error {
	c := connector.pool.Get()
	defer c.Close()

	var readingErr error

	key := fmt.Sprintf("moira-metric-last-check:%s", triggerId)
	lastCheckString, readingErr := redis.String(c.Do("GET", key))
	if readingErr != nil {
		if readingErr != redis.ErrNil {
			return nil
		}
	}

	//todo кажется, здесь есть баг, связанный с конкурентностью запросов
	for readingErr != redis.ErrNil {
		var lastCheck = moira.CheckData{}
		err := json.Unmarshal([]byte(lastCheckString), &lastCheck)
		if err != nil {
			return fmt.Errorf("Failed to parse lastCheck json %s: %s", lastCheckString, err.Error())
		}
		metricsCheck := lastCheck.Metrics
		if metricsCheck != nil && len(metricsCheck) > 0 {
			for metric, value := range metrics {
				data, ok := metricsCheck[metric]
				if !ok {
					data = moira.MetricData{}
				}
				data.Maintenance = &value
				metricsCheck[metric] = data
			}
		}
		newLastCheck, err := json.Marshal(lastCheck)
		if err != nil {
			return err
		}

		prev, readingErr := redis.String(c.Do("GETSET", key, newLastCheck))
		if readingErr != nil {
			if readingErr != redis.ErrNil {
				return readingErr
			}
		}
		if prev == lastCheckString {
			break
		}
		lastCheckString = prev
	}

	return nil
}

func (connector *DbConnector) GetNotifications(start, end int64) ([]*moira.ScheduledNotification, int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("ZRANGE", "moira-notifier-notifications", start, end)
	c.Send("ZCARD", "moira-notifier-notifications")
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	if len(rawResponse) == 0 {
		return make([]*moira.ScheduledNotification, 0), 0, nil
	}
	total, err := redis.Int(rawResponse[1], nil)
	if err != nil {
		return nil, 0, err
	}
	notifications, err := connector.convertNotifications(rawResponse[0])
	if err != nil {
		return nil, 0, err
	}
	return notifications, int64(total), nil
}

func (connector *DbConnector) RemoveNotification(notificationKey string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	notifications, _, err := connector.GetNotifications(0, -1)
	if err != nil {
		return 0, err
	}
	//todo кажется, что здесь баг, потомучто удаляется не все нотификации, а только первая попавшаяся
	for _, notification := range notifications {
		timestamp := strconv.FormatInt(notification.Timestamp, 10)
		contactId := notification.Contact.ID
		subId := moira.UseString(notification.Event.SubscriptionID)
		idstr := strings.Join([]string{timestamp, contactId, subId}, "")
		if idstr == notificationKey {
			notificationString, err := json.Marshal(notification)
			if err != nil {
				return 0, err
			}
			result, err := redis.Int64(c.Do("ZREM", "moira-notifier-notifications", notificationString))
			if err != nil {
				return 0, fmt.Errorf("Failed to remove notifier-notification: %s", err.Error())
			}
			return result, nil
		}
	}
	return 0, nil
}

type TriggerStorageElement struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Desc            *string             `json:"desc,omitempty"`
	Targets         []string            `json:"targets"`
	WarnValue       *float64            `json:"warn_value"`
	ErrorValue      *float64            `json:"error_value"`
	Tags            []string            `json:"tags"`
	TtlState        *string             `json:"ttl_state,omitempty"`
	Schedule        *moira.ScheduleData `json:"sched,omitempty"`
	Expression      *string             `json:"expression,omitempty"`
	Patterns        []string            `json:"patterns"`
	IsSimpleTrigger bool                `json:"is_simple_trigger"`
	Ttl             *string             `json:"ttl"`
}

func toTrigger(storageElement *TriggerStorageElement, triggerId string) *moira.Trigger {
	return &moira.Trigger{
		ID:              triggerId,
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
		Ttl:             getTriggerTtl(storageElement.Ttl),
	}
}

func toTriggerStorageElement(trigger *moira.Trigger, triggerId string) *TriggerStorageElement {
	return &TriggerStorageElement{
		ID:              triggerId,
		Name:            trigger.Name,
		Desc:            trigger.Desc,
		Targets:         trigger.Targets,
		WarnValue:       trigger.WarnValue,
		ErrorValue:      trigger.ErrorValue,
		Tags:            trigger.Tags,
		TtlState:        trigger.TtlState,
		Schedule:        trigger.Schedule,
		Expression:      trigger.Expression,
		Patterns:        trigger.Patterns,
		IsSimpleTrigger: trigger.IsSimpleTrigger,
		Ttl:             getTriggerTtlString(trigger.Ttl),
	}
}

func getTriggerTtl(ttlstr *string) *int64 {
	if ttlstr == nil {
		return nil
	}
	ttl, _ := strconv.ParseInt(*ttlstr, 10, 64)
	return &ttl
}

func getTriggerTtlString(ttl *int64) *string {
	if ttl == nil {
		return nil
	}
	ttlString := fmt.Sprintf("%v", *ttl)
	return &ttlString
}

func (connector *DbConnector) convertTriggerWithTags(triggerInterface interface{}, triggerTagsInterface interface{}, triggerId string) (*TriggerStorageElement, error) {
	trigger := &TriggerStorageElement{}
	triggerBytes, err := redis.Bytes(triggerInterface, nil)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("Error getting trigger bytes, id: %s, error: %s", triggerId, err.Error())
	}
	if err := json.Unmarshal(triggerBytes, trigger); err != nil {
		return nil, fmt.Errorf("Failed to parse trigger json %s: %s", triggerBytes, err.Error())
	}
	triggerTags, err := redis.Strings(triggerTagsInterface, nil)
	if err != nil {
		connector.logger.Errorf("Error getting trigger-tags, id: %s, error: %s", triggerId, err.Error())
	}
	if triggerTags != nil && len(triggerTags) > 0 {
		trigger.Tags = triggerTags
	}
	return trigger, nil
}
