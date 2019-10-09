package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

var eventsTTL int64 = 3600 * 24 * 30

// GetNotificationEvents gets NotificationEvents by given triggerID and interval
func (connector *DbConnector) GetNotificationEvents(triggerID string, start int64, size int64) ([]*moira.NotificationEvent, error) {
	c := connector.pool.Get()
	defer c.Close()

	eventsData, err := reply.Events(c.Do("ZREVRANGE", triggerEventsKey(triggerID), start, start+size))

	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.NotificationEvent, 0), nil
		}
		return nil, fmt.Errorf("failed to get range for trigger events, triggerID: %s, error: %s", triggerID, err.Error())
	}

	return eventsData, nil
}

// PushNotificationEvent adds new NotificationEvent to events list and to given triggerID events list and deletes events who are older than 30 days
// If ui=true, then add to ui events list
func (connector *DbConnector) PushNotificationEvent(event *moira.NotificationEvent, ui bool) error {
	eventBytes, err := reply.GetEventBytes(*event)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("LPUSH", notificationEventsList, eventBytes)
	if event.TriggerID != "" {
		c.Send("ZADD", triggerEventsKey(event.TriggerID), event.Timestamp, eventBytes)
		c.Send("ZREMRANGEBYSCORE", triggerEventsKey(event.TriggerID), "-inf", time.Now().Unix()-eventsTTL)
	}
	if ui {
		c.Send("LPUSH", notificationEventsUIList, eventBytes)
		c.Send("LTRIM", notificationEventsUIList, 0, 100)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetNotificationEventCount returns planned notifications count from given timestamp
func (connector *DbConnector) GetNotificationEventCount(triggerID string, from int64) int64 {
	c := connector.pool.Get()
	defer c.Close()

	count, _ := redis.Int64(c.Do("ZCOUNT", triggerEventsKey(triggerID), from, "+inf"))
	return count
}

// FetchNotificationEvent waiting for event in events list
func (connector *DbConnector) FetchNotificationEvent() (moira.NotificationEvent, error) {
	c := connector.pool.Get()
	defer c.Close()

	var event moira.NotificationEvent

	rawRes, err := c.Do("BRPOP", notificationEventsList, 1)
	if err != nil {
		return event, fmt.Errorf("failed to fetch event: %s", err.Error())
	}
	if rawRes == nil {
		return event, database.ErrNil
	}
	var (
		eventBytes []byte
		key        []byte
	)
	res, _ := redis.Values(rawRes, nil)
	if _, err = redis.Scan(res, &key, &eventBytes); err != nil {
		return event, fmt.Errorf("failed to parse event: %s", err.Error())
	}
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return event, fmt.Errorf("failed to parse event json %s: %s", eventBytes, err.Error())
	}
	if event.Values == nil { //TODO(litleleprikon): remove in moira v2.8.0. Compatibility with moira < v2.6.0
		event.Values = make(map[string]float64)
	}
	if event.Value != nil {
		event.Values["t1"] = *event.Value
		event.Value = nil
	}
	return event, nil
}

// RemoveAllNotificationEvents removes all notification events from database
func (connector *DbConnector) RemoveAllNotificationEvents() error {
	c := connector.pool.Get()
	defer c.Close()

	if _, err := c.Do("DEL", notificationEventsList); err != nil {
		return fmt.Errorf("failed to remove %s: %s", notificationEventsList, err.Error())
	}

	return nil
}

var notificationEventsList = "moira-trigger-events"
var notificationEventsUIList = "moira-trigger-events-ui"

func triggerEventsKey(triggerID string) string {
	return "moira-trigger-events:" + triggerID
}
