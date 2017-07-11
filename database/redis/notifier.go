package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
)

//DbConnector contains redis pool
type DbConnector struct {
	pool    *redis.Pool
	logger  moira_alert.Logger
	metrics *graphite.NotifierMetrics
}

// FetchEvent waiting for event from Db
func (connector *DbConnector) FetchEvent() (*moira_alert.EventData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var event moira_alert.EventData

	rawRes, err := c.Do("BRPOP", "moira-trigger-events", 1)
	if err != nil {
		connector.logger.Warningf("Failed to wait for event: %s", err.Error())
		time.Sleep(time.Second * 5)
		return nil, nil
	}
	if rawRes != nil {
		var (
			eventBytes []byte
			key        []byte
		)
		res, _ := redis.Values(rawRes, nil)
		if _, err = redis.Scan(res, &key, &eventBytes); err != nil {
			connector.logger.Warningf("Failed to parse event: %s", err.Error())
			return nil, err
		}
		if err := json.Unmarshal(eventBytes, &event); err != nil {
			connector.logger.Error(fmt.Sprintf("Failed to parse event json %s: %s", eventBytes, err.Error()))
			return nil, err
		}
		return &event, nil
	}

	return nil, nil
}

// GetTrigger returns trigger data
func (connector *DbConnector) GetTrigger(id string) (moira_alert.TriggerData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var trigger moira_alert.TriggerData

	triggerString, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-trigger:%s", id)))
	if err != nil {
		return trigger, fmt.Errorf("Failed to get trigger data for id %s: %s", id, err.Error())
	}
	if err := json.Unmarshal(triggerString, &trigger); err != nil {
		return trigger, fmt.Errorf("Failed to parse trigger json %s: %s", triggerString, err.Error())
	}

	return trigger, nil
}

// GetTriggerTags returns trigger tags
func (connector *DbConnector) GetTriggerTags(triggerID string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	var tags []string

	values, err := redis.Values(c.Do("SMEMBERS", fmt.Sprintf("moira-trigger-tags:%s", triggerID)))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags for trigger id %s: %s", triggerID, err.Error())
	}
	if err := redis.ScanSlice(values, &tags); err != nil {
		return nil, fmt.Errorf("Failed to retrieve tags for trigger id %s: %s", triggerID, err.Error())
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("No tags found for trigger id %s", triggerID)
	}
	return tags, nil
}

// GetTagsSubscriptions returns all subscriptions for given tags list
func (connector *DbConnector) GetTagsSubscriptions(tags []string) ([]moira_alert.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	connector.logger.Debugf("Getting tags %v subscriptions", tags)

	tagKeys := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		tagKeys = append(tagKeys, fmt.Sprintf("moira-tag-subscriptions:%s", tag))
	}
	values, err := redis.Values(c.Do("SUNION", tagKeys...))
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for tags %v: %s", tags, err.Error())
	}
	var subscriptions []string
	if err := redis.ScanSlice(values, &subscriptions); err != nil {
		return nil, fmt.Errorf("Failed to retrieve subscriptions for tags %v: %s", tags, err.Error())
	}
	if len(subscriptions) == 0 {
		connector.logger.Debugf("No subscriptions found for tag set %v", tags)
		return make([]moira_alert.SubscriptionData, 0, 0), nil
	}

	var subscriptionsData []moira_alert.SubscriptionData
	for _, id := range subscriptions {
		sub, err := connector.GetSubscription(id)
		if err != nil {
			continue
		}
		subscriptionsData = append(subscriptionsData, sub)
	}
	return subscriptionsData, nil
}

// GetSubscription returns subscription data by given id
func (connector *DbConnector) GetSubscription(id string) (moira_alert.SubscriptionData, error) {
	c := connector.pool.Get()
	defer c.Close()

	sub := moira_alert.SubscriptionData{
		ThrottlingEnabled: true,
	}
	subscriptionString, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-subscription:%s", id)))
	if err != nil {
		connector.metrics.SubsMalformed.Mark(1)
		return sub, fmt.Errorf("Failed to get subscription data for id %s: %s", id, err.Error())
	}
	if err := json.Unmarshal(subscriptionString, &sub); err != nil {
		connector.metrics.SubsMalformed.Mark(1)
		return sub, fmt.Errorf("Failed to parse subscription json %s: %s", subscriptionString, err.Error())
	}
	sub.ID = id
	return sub, nil
}

// GetContact returns contact data by given id
func (connector *DbConnector) GetContact(id string) (moira_alert.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var contact moira_alert.ContactData

	contactString, err := redis.Bytes(c.Do("GET", fmt.Sprintf("moira-contact:%s", id)))
	if err != nil {
		return contact, fmt.Errorf("Failed to get contact data for id %s: %s", id, err.Error())
	}
	if err := json.Unmarshal(contactString, &contact); err != nil {
		return contact, fmt.Errorf("Failed to parse contact json %s: %s", contactString, err.Error())
	}
	contact.ID = id
	return contact, nil
}

// GetContacts returns full contact list
func (connector *DbConnector) GetContacts() ([]moira_alert.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var result []moira_alert.ContactData
	keys, err := redis.Strings(c.Do("KEYS", "moira-contact:*"))
	if err != nil {
		return result, err
	}
	for _, key := range keys {
		key = key[14:]
		contact, _ := connector.GetContact(key)
		result = append(result, contact)
	}
	return result, err
}

// SetContact store contact information
func (connector *DbConnector) SetContact(contact *moira_alert.ContactData) error {
	id := contact.ID
	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("SET", fmt.Sprintf("moira-contact:%s", id), contactString); err != nil {
		return err
	}
	return nil
}

// AddNotification store notification at given timestamp
func (connector *DbConnector) AddNotification(notification *moira_alert.ScheduledNotification) error {

	notificationString, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()

	if _, err := c.Do("ZADD", "moira-notifier-notifications", notification.Timestamp, notificationString); err != nil {
		return err
	}

	return nil
}

// GetTriggerThrottlingTimestamps get throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) GetTriggerThrottlingTimestamps(triggerID string) (time.Time, time.Time) {
	c := connector.pool.Get()
	defer c.Close()

	next, _ := redis.Int64(c.Do("GET", fmt.Sprintf("moira-notifier-next:%s", triggerID)))
	beginning, _ := redis.Int64(c.Do("GET", fmt.Sprintf("moira-notifier-throttling-beginning:%s", triggerID)))

	return time.Unix(next, 0), time.Unix(beginning, 0)
}

// GetTriggerEventsCount retuns planned notifications count from given timestamp
func (connector *DbConnector) GetTriggerEventsCount(triggerID string, from int64) int64 {
	c := connector.pool.Get()
	defer c.Close()

	eventsKey := fmt.Sprintf("moira-trigger-events:%s", triggerID)
	count, _ := redis.Int64(c.Do("ZCOUNT", eventsKey, from, "+inf"))
	return count
}

// SetTriggerThrottlingTimestamp store throttling or scheduled notifications delay for given triggerID
func (connector *DbConnector) SetTriggerThrottlingTimestamp(triggerID string, next time.Time) error {
	c := connector.pool.Get()
	defer c.Close()
	if _, err := c.Do("SET", fmt.Sprintf("moira-notifier-next:%s", triggerID), next.Unix()); err != nil {
		return err
	}
	return nil
}

// GetNotifications fetch notifications by given timestamp
func (connector *DbConnector) GetNotifications(to int64) ([]*moira_alert.ScheduledNotification, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZRANGEBYSCORE", "moira-notifier-notifications", "-inf", to)
	c.Send("ZREMRANGEBYSCORE", "moira-notifier-notifications", "-inf", to)
	redisRawResponse, err := c.Do("EXEC")
	if err != nil {
		return nil, err
	}

	redisResponse, err := redis.Values(redisRawResponse, nil)
	if err != nil {
		return nil, err
	}

	return connector.convertNotifications(redisResponse[0])
}

// GetMetricsCount - return metrics count received by Moira-Cache
func (connector *DbConnector) GetMetricsCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", "moira-selfstate:metrics-heartbeat"))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// GetChecksCount - return checks count by Moira-Checker
func (connector *DbConnector) GetChecksCount() (int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	ts, err := redis.Int64(c.Do("GET", "moira-selfstate:checks-counter"))
	if err == redis.ErrNil {
		return 0, nil
	}
	return ts, err
}

// ConvertNotifications extracts ScheduledNotification from redis response
func (connector *DbConnector) convertNotifications(redisResponse interface{}) ([]*moira_alert.ScheduledNotification, error) {

	notificationStrings, err := redis.Strings(redisResponse, nil)
	if err != nil {
		return nil, err
	}

	notifications := make([]*moira_alert.ScheduledNotification, 0, len(notificationStrings))

	for _, notificationString := range notificationStrings {
		notification := &moira_alert.ScheduledNotification{}
		if err := json.Unmarshal([]byte(notificationString), notification); err != nil {
			connector.logger.Warningf("Failed to parse scheduled json notification %s: %s", notificationString, err.Error())
			continue
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}
