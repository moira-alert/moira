package redis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira/notifier"

	"github.com/gomodule/redigo/redis"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

const (
	transactionTriesLimit     = 10
	transactionHeuristicLimit = 10000
)

// Custom error for transaction error
type transactionError struct{}

func (e transactionError) Error() string {
	return "Transaction Error"
}

// Drops all notifications with last timestamp
func limitNotifications(notifications []*moira.ScheduledNotification) []*moira.ScheduledNotification {
	if len(notifications) == 0 {
		return notifications
	}
	i := len(notifications) - 1
	lastTs := notifications[i].Timestamp

	for ; i >= 0; i-- {
		if notifications[i].Timestamp != lastTs {
			break
		}
	}

	if i == -1 {
		return notifications
	}

	return notifications[:i+1]
}

// GetNotifications gets ScheduledNotifications in given range and full range
func (connector *DbConnector) GetNotifications(start, end int64) ([]*moira.ScheduledNotification, int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZRANGE", notifierNotificationsKey, start, end)
	c.Send("ZCARD", notifierNotificationsKey)
	rawResponse, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to EXEC: %s", err.Error())
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

// RemoveAllNotifications delete all notifications
func (connector *DbConnector) RemoveAllNotifications() error {
	c := connector.pool.Get()
	defer c.Close()

	if _, err := c.Do("DEL", notifierNotificationsKey); err != nil {
		return fmt.Errorf("failed to remove %s: %s", notifierNotificationsKey, err.Error())
	}

	return nil
}

// RemoveNotification delete notifications by key = timestamp + contactID + subID
func (connector *DbConnector) RemoveNotification(notificationKey string) (int64, error) {
	notifications, _, err := connector.GetNotifications(0, -1)
	if err != nil {
		return 0, err
	}

	foundNotifications := make([]*moira.ScheduledNotification, 0)
	for _, notification := range notifications {
		timestamp := strconv.FormatInt(notification.Timestamp, 10)
		contactID := notification.Contact.ID
		subID := moira.UseString(notification.Event.SubscriptionID)
		idstr := strings.Join([]string{timestamp, contactID, subID}, "")
		if idstr == notificationKey {
			foundNotifications = append(foundNotifications, notification)
		}
	}
	return connector.removeNotifications(foundNotifications)
}

func (connector *DbConnector) removeNotifications(notifications []*moira.ScheduledNotification) (int64, error) {
	if len(notifications) == 0 {
		return 0, nil
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, notification := range notifications {
		notificationString, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return 0, err
		}
		c.Send("ZREM", notifierNotificationsKey, notificationString)
	}
	response, err := redis.Ints(c.Do("EXEC"))
	if err != nil {
		return 0, fmt.Errorf("failed to remove notifier-notification: %s", err.Error())
	}
	total := 0
	for _, val := range response {
		total += val
	}
	return int64(total), nil
}

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) FetchNotifications(to int64, limit int64) ([]*moira.ScheduledNotification, error) {

	if limit == 0 {
		return nil, fmt.Errorf("limit mustn't be 0")
	}

	// No limit
	if limit == notifier.NotificationsLimitUnlimited {
		return connector.fetchNotificationsNoLimit(to)
	}

	count, err := connector.notificationsCount(to)
	if err != nil {
		return nil, err
	}

	// Hope count will be not greater then limit when we call fetchNotificationsNoLimit
	if limit > transactionHeuristicLimit && count < limit/2 {
		return connector.fetchNotificationsNoLimit(to)
	}

	return connector.fetchNotificationsWithLimit(to, limit)
}

func (connector *DbConnector) notificationsCount(to int64) (int64, error) {
	c := connector.pool.Get()
	defer c.Close()

	count, err := redis.Int64(c.Do("ZCOUNT", notifierNotificationsKey, "-inf", to))

	if err != nil {
		return 0, fmt.Errorf("failed to ZCOUNT to notificationsCount: %w", err)
	}

	return count, nil
}

// fetchNotificationsWithLimit reads and drops notifications from DB with limit
func (connector *DbConnector) fetchNotificationsWithLimit(to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	// fetchNotifecationsWithLimitDo uses WATCH, so transaction may fail and will retry it
	// see https://redis.io/topics/transactions

	for i := 0; i < transactionTriesLimit; i++ {
		res, err := connector.fetchNotificationsWithLimitDo(to, limit)

		if err == nil {
			return res, nil
		}

		if !errors.As(err, &transactionError{}) {
			return nil, err
		}

		time.Sleep(200 * time.Millisecond)
	}

	return nil, fmt.Errorf("Transaction tries limit exceeded")
}

// same as fetchNotificationsWithLimit, but only once
func (connector *DbConnector) fetchNotificationsWithLimitDo(to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	// see https://redis.io/topics/transactions

	c := connector.pool.Get()
	defer c.Close()

	// start optimistic transaction and get notifications with LIMIT
	c.Send("WATCH", notifierNotificationsKey)
	response, err := redis.Values(c.Do("ZRANGEBYSCORE", notifierNotificationsKey, "-inf", to, "LIMIT", 0, limit))
	if err != nil {
		return nil, fmt.Errorf("failed to ZRANGEBYSCORE: %s", err)
	}

	if len(response) == 0 {
		return make([]*moira.ScheduledNotification, 0), nil
	}

	notifications, err := reply.Notifications(response, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}

	// ZRANGEBYSCORE with LIMIT may return not all notification with last timestamp
	// (e.g. if we have notifications with timestamps [1, 2, 3, 3, 3] and limit == 3)
	// but ZREMRANGEBYSCORE does not have LIMIT, so will delete all notifications with last timestamp
	// (ts 3 in our example) and then run ZRANGEBYSCORE with our new last timestamp (2 in our example).

	notificationsLimited := limitNotifications(notifications)
	lastTs := notificationsLimited[len(notificationsLimited)-1].Timestamp

	if len(notifications) == len(notificationsLimited) {
		// this means that all notifications have same timestamp,
		// we hope that all notifications with same timestamp should fit our memory
		c.Send("UNWATCH")
		return connector.fetchNotificationsNoLimit(lastTs)
	}

	c.Send("MULTI")
	c.Send("ZREMRANGEBYSCORE", notifierNotificationsKey, "-inf", lastTs)
	deleteCount, errDelete := redis.Values(c.Do("EXEC"))
	if errDelete != nil {
		return nil, fmt.Errorf("failed to EXEC: %w", errDelete)
	}

	// someone has changed notifierNotificationsKey while we do our job
	// and transaction fail (no notifications were deleted) :(
	if deleteCount == nil {
		tr := transactionError{}
		return nil, &tr
	}

	return notificationsLimited, nil
}

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) fetchNotificationsNoLimit(to int64) ([]*moira.ScheduledNotification, error) {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("ZRANGEBYSCORE", notifierNotificationsKey, "-inf", to)
	c.Send("ZREMRANGEBYSCORE", notifierNotificationsKey, "-inf", to)
	response, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}
	if len(response) == 0 {
		return make([]*moira.ScheduledNotification, 0), nil
	}
	return reply.Notifications(response[0], nil)
}

// AddNotification store notification at given timestamp
func (connector *DbConnector) AddNotification(notification *moira.ScheduledNotification) error {
	bytes, err := reply.GetNotificationBytes(*notification)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()

	_, err = c.Do("ZADD", notifierNotificationsKey, notification.Timestamp, bytes)
	if err != nil {
		return fmt.Errorf("failed to add scheduled notification: %s, error: %s", string(bytes), err.Error())
	}
	return err
}

// AddNotifications store notification at given timestamp
func (connector *DbConnector) AddNotifications(notifications []*moira.ScheduledNotification, timestamp int64) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	for _, notification := range notifications {
		bytes, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return err
		}
		c.Send("ZADD", notifierNotificationsKey, timestamp, bytes)
	}
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

var notifierNotificationsKey = "moira-notifier-notifications"
