package redis

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

var eventsTTL int64 = 3600 * 24 * 30

// GetNotificationEvents gets NotificationEvents by given triggerID and interval.
func (connector *DbConnector) GetNotificationEvents(triggerID string, start int64, size int64) ([]*moira.NotificationEvent, error) {
	ctx := connector.context
	c := *connector.client

	eventsData, err := reply.Events(c.ZRevRange(ctx, triggerEventsKey(triggerID), start, start+size))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return make([]*moira.NotificationEvent, 0), nil
		}
		return nil, fmt.Errorf("failed to get range for trigger events, triggerID: %s, error: %s", triggerID, err.Error())
	}

	return eventsData, nil
}

// PushNotificationEvent adds new NotificationEvent to events list and to given triggerID events list and deletes events who are older than 30 days.
// If ui=true, then add to ui events list.
func (connector *DbConnector) PushNotificationEvent(event *moira.NotificationEvent, ui bool) error {
	eventBytes, err := reply.GetEventBytes(*event)
	if err != nil {
		return err
	}

	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe.LPush(ctx, notificationEventsList, eventBytes)
	if event.TriggerID != "" {
		z := &redis.Z{Score: float64(event.Timestamp), Member: eventBytes}
		to := int(time.Now().Unix() - eventsTTL)

		pipe.ZAdd(ctx, triggerEventsKey(event.TriggerID), z)
		pipe.ZRemRangeByScore(ctx, triggerEventsKey(event.TriggerID), "-inf", strconv.Itoa(to))
	}

	if ui {
		pipe.LPush(ctx, notificationEventsUIList, eventBytes)
		pipe.LTrim(ctx, notificationEventsUIList, 0, 100)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

// GetNotificationEventCount returns planned notifications count from given timestamp.
func (connector *DbConnector) GetNotificationEventCount(triggerID string, from int64) int64 {
	ctx := connector.context
	c := *connector.client

	count, _ := c.ZCount(ctx, triggerEventsKey(triggerID), strconv.FormatInt(from, 10), "+inf").Result()
	return count
}

// FetchNotificationEvent waiting for event in events list.
func (connector *DbConnector) FetchNotificationEvent() (moira.NotificationEvent, error) {
	var event moira.NotificationEvent
	ctx := connector.context
	c := *connector.client

	response := c.BRPop(ctx, time.Second, notificationEventsList)
	err := response.Err()

	if errors.Is(err, redis.Nil) {
		return event, database.ErrNil
	}

	if err != nil {
		return event, fmt.Errorf("failed to fetch event: %s", err.Error())
	}

	event, _ = reply.BRPopToEvent(response)

	if event.Values == nil { // TODO(litleleprikon): remove in moira v2.8.0. Compatibility with moira < v2.6.0
		event.Values = make(map[string]float64)
	}

	if event.Value != nil {
		event.Values["t1"] = *event.Value
		event.Value = nil
	}

	return event, nil
}

// RemoveAllNotificationEvents removes all notification events from database.
func (connector *DbConnector) RemoveAllNotificationEvents() error {
	ctx := connector.context
	c := *connector.client

	if _, err := c.Del(ctx, notificationEventsList).Result(); err != nil {
		return fmt.Errorf("failed to remove %s: %s", notificationEventsList, err.Error())
	}

	return nil
}

var (
	notificationEventsList   = "moira-trigger-events"
	notificationEventsUIList = "moira-trigger-events-ui"
)

func triggerEventsKey(triggerID string) string {
	return "moira-trigger-events:" + triggerID
}
