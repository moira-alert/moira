package redis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira/notifier"

	"github.com/go-redis/redis/v8"

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
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe.ZRange(ctx, notifierNotificationsKey, start, end)
	pipe.ZCard(ctx, notifierNotificationsKey)
	response, err := pipe.Exec(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	if len(response) == 0 {
		return make([]*moira.ScheduledNotification, 0), 0, nil
	}

	total, err := response[1].(*redis.IntCmd).Result()
	if err != nil {
		return nil, 0, err
	}

	notifications, err := reply.Notifications(response[0].(*redis.StringSliceCmd))
	if err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// RemoveAllNotifications delete all notifications
func (connector *DbConnector) RemoveAllNotifications() error {
	ctx := connector.context
	c := *connector.client

	if _, err := c.Del(ctx, notifierNotificationsKey).Result(); err != nil {
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

	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	for _, notification := range notifications {
		notificationString, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return 0, err
		}
		pipe.ZRem(ctx, notifierNotificationsKey, notificationString)
	}
	response, err := pipe.Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to remove notifier-notification: %s", err.Error())
	}

	total := int64(0)
	for _, val := range response {
		intVal, _ := val.(*redis.IntCmd).Result()
		total += intVal
	}

	return total, nil
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
	ctx := connector.context
	c := *connector.client

	count, err := c.ZCount(ctx, notifierNotificationsKey, "-inf", strconv.FormatInt(to, 10)).Result()

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

		time.Sleep(200 * time.Millisecond) //nolint
	}

	return nil, fmt.Errorf("Transaction tries limit exceeded")
}

// same as fetchNotificationsWithLimit, but only once
func (connector *DbConnector) fetchNotificationsWithLimitDo(to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	// see https://redis.io/topics/transactions

	ctx := connector.context
	c := *connector.client

	// start optimistic transaction and get notifications with LIMIT
	var response *redis.StringSliceCmd

	err := c.Watch(ctx, func(tx *redis.Tx) error {
		rng := &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10), Offset: 0, Count: limit}
		response = tx.ZRangeByScore(ctx, notifierNotificationsKey, rng)

		return response.Err()
	}, notifierNotificationsKey)

	if err != nil {
		return nil, fmt.Errorf("failed to ZRANGEBYSCORE: %s", err)
	}

	notifications, err := reply.Notifications(response)
	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}

	if len(notifications) == 0 {
		return notifications, nil
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
		return connector.fetchNotificationsNoLimit(lastTs)
	}

	pipe := c.TxPipeline()
	pipe.ZRemRangeByScore(ctx, notifierNotificationsKey, "-inf", strconv.FormatInt(lastTs, 10))
	rangeResponse, errDelete := pipe.Exec(ctx)

	if errDelete != nil {
		return nil, fmt.Errorf("failed to EXEC: %w", errDelete)
	}

	// someone has changed notifierNotificationsKey while we do our job
	// and transaction fail (no notifications were deleted) :(
	deleteCount, errConvert := rangeResponse[0].(*redis.IntCmd).Result()
	if errConvert != nil || deleteCount == 0 {
		return nil, &transactionError{}
	}

	return notificationsLimited, nil
}

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) fetchNotificationsNoLimit(to int64) ([]*moira.ScheduledNotification, error) {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	pipe.ZRangeByScore(ctx, notifierNotificationsKey, &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10)})
	pipe.ZRemRangeByScore(ctx, notifierNotificationsKey, "-inf", strconv.FormatInt(to, 10))
	response, err := pipe.Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to EXEC: %s", err)
	}

	return reply.Notifications(response[0].(*redis.StringSliceCmd))
}

// AddNotification store notification at given timestamp
func (connector *DbConnector) AddNotification(notification *moira.ScheduledNotification) error {
	bytes, err := reply.GetNotificationBytes(*notification)
	if err != nil {
		return err
	}

	ctx := connector.context
	c := *connector.client

	z := &redis.Z{Score: float64(notification.Timestamp), Member: bytes}
	_, err = c.ZAdd(ctx, notifierNotificationsKey, z).Result()
	if err != nil {
		return fmt.Errorf("failed to add scheduled notification: %s, error: %s", string(bytes), err.Error())
	}

	return err
}

// AddNotifications store notification at given timestamp
func (connector *DbConnector) AddNotifications(notifications []*moira.ScheduledNotification, timestamp int64) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()
	for _, notification := range notifications {
		bytes, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return err
		}

		z := &redis.Z{Score: float64(timestamp), Member: bytes}
		pipe.ZAdd(ctx, notifierNotificationsKey, z)
	}
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

var notifierNotificationsKey = "moira-notifier-notifications"
