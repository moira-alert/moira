package redis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/garyburd/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetNotifications gets ScheduledNotifications in given range and full range
func (connector *DbConnector) GetNotifications(start, end int64) ([]*moira.ScheduledNotification, int64, error) {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	c.Send("ZRANGE", notifierNotificationsKey, start, end)
	c.Send("ZCARD", notifierNotificationsKey)
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	if len(rawResponse) == 0 {
		return make([]*moira.ScheduledNotification, 0), 0, nil
	}
	total, err := redis.Int64(rawResponse[1], nil)
	if err != nil {
		return nil, 0, err
	}
	notifications, err := reply.Notifications(rawResponse[0], nil)
	if err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

// RemoveNotification delete notifications by key = timestamp + contactID + subID
func (connector *DbConnector) RemoveNotification(notificationKey string) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	notifications, _, err := connector.GetNotifications(0, -1)
	if err != nil {
		return 0, err
	}

	c.Send("MULTI")

	for _, notification := range notifications {
		timestamp := strconv.FormatInt(notification.Timestamp, 10)
		contactID := notification.Contact.ID
		subID := moira.UseString(notification.Event.SubscriptionID)
		idstr := strings.Join([]string{timestamp, contactID, subID}, "")
		if idstr == notificationKey {
			notificationString, err2 := json.Marshal(notification)
			if err2 != nil {
				return 0, err2
			}
			c.Send("ZREM", notifierNotificationsKey, notificationString)
		}
	}
	response, err := redis.Ints(c.Do("EXEC"))
	if err != nil {
		return 0, fmt.Errorf("Failed to remove notifier-notification: %s", err.Error())
	}
	total := 0
	for _, val := range response {
		total += val
	}
	return int64(total), nil
}

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) FetchNotifications(to int64) ([]*moira.ScheduledNotification, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZRANGEBYSCORE", notifierNotificationsKey, "-inf", to)
	c.Send("ZREMRANGEBYSCORE", notifierNotificationsKey, "-inf", to)
	response, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("Failed to EXEC: %s", err)
	}
	if len(response) == 0 {
		return make([]*moira.ScheduledNotification, 0), nil
	}
	return reply.Notifications(response[0], nil)
}

// AddNotification store notification at given timestamp
func (connector *DbConnector) AddNotification(notification *moira.ScheduledNotification) error {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	_, err = c.Do("ZADD", notifierNotificationsKey, notification.Timestamp, bytes)
	if err != nil {
		return fmt.Errorf("Failed to add scheduled notification: %s, error: %s", string(bytes), err.Error())
	}
	return err
}

// AddNotifications store notification at given timestamp
func (connector *DbConnector) AddNotifications(notifications []*moira.ScheduledNotification, timestamp int64) error {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, notification := range notifications {
		bytes, err := json.Marshal(notification)
		if err != nil {
			return err
		}
		c.Send("ZADD", notifierNotificationsKey, timestamp, bytes)
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

var notifierNotificationsKey = "moira-notifier-notifications"
