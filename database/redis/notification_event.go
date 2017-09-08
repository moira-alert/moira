package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

var eventsTTL int64 = 3600 * 24 * 30

// GetNotificationEvents gets NotificationEvents by given triggerID and interval
func (connector *DbConnector) GetNotificationEvents(triggerID string, start int64, size int64) ([]*moira.NotificationEvent, error) {
	c := connector.pool.Get()
	defer c.Close()

	eventsData, err := reply.Events(c.Do("ZREVRANGE", moiraTriggerEvents(triggerID), start, start+size))

	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.NotificationEvent, 0), nil
		}
		return nil, fmt.Errorf("Failed to get range for trigger events, triggerID: %s, error: %s", triggerID, err.Error())
	}

	return eventsData, nil
}

// PushNotificationEvent adds new NotificationEvent to events list and to given triggerID events list and deletes events who are older than 30 days
// If ui=true, then add to ui events list
func (connector *DbConnector) PushNotificationEvent(event *moira.NotificationEvent, ui bool) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("LPUSH", moiraEventsList, eventBytes)
	if event.TriggerID != "" {
		c.Send("ZADD", moiraTriggerEvents(event.TriggerID), event.Timestamp, eventBytes)
		c.Send("ZREMRANGEBYSCORE", moiraTriggerEvents(event.TriggerID), "-inf", time.Now().Unix()-eventsTTL)
	}
	if ui {
		c.Send("LPUSH", moiraEventsUIList, eventBytes)
		c.Send("LTRIM", moiraEventsUIList, 0, 100)
	}
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetNotificationEventCount returns planned notifications count from given timestamp
func (connector *DbConnector) GetNotificationEventCount(triggerID string, from int64) int64 {
	c := connector.pool.Get()
	defer c.Close()

	count, _ := redis.Int64(c.Do("ZCOUNT", moiraTriggerEvents(triggerID), from, "+inf"))
	return count
}

// FetchNotificationEvent waiting for event in events list
func (connector *DbConnector) FetchNotificationEvent() (moira.NotificationEvent, error) {
	c := connector.pool.Get()
	defer c.Close()

	var event moira.NotificationEvent

	rawRes, err := c.Do("BRPOP", moiraEventsList, 1)
	if err != nil {
		return event, fmt.Errorf("Failed to fetch event: %s", err.Error())
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
		return event, fmt.Errorf("Failed to parse event: %s", err.Error())
	}
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return event, fmt.Errorf("Failed to parse event json %s: %s", eventBytes, err.Error())
	}
	return event, nil
}

var moiraEventsList = "moira-trigger-events"
var moiraEventsUIList = "moira-trigger-events-ui"

func moiraTriggerEvents(triggerID string) string {
	return fmt.Sprintf("moira-trigger-events:%s", triggerID)
}
