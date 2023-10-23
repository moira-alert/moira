package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moira-alert/moira/notifier"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

const (
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

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) FetchNotificationsNoLimitOther(to int64) ([]*moira.ScheduledNotification, error) {
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

	return connector.removeNotifications(connector.context, (*connector.client).TxPipeline(), foundNotifications)
}

func (connector *DbConnector) removeNotifications(ctx context.Context, pipe redis.Pipeliner, notifications []*moira.ScheduledNotification) (int64, error) {
	if len(notifications) == 0 {
		return 0, nil
	}

	for _, notification := range notifications {
		notificationString, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return 0, err
		}
		pipe.ZRem(ctx, notifierNotificationsKey, notificationString)
	}
	response, err := pipe.Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to remove notifications: %w", err)
	}

	total := int64(0)
	for _, val := range response {
		intVal, _ := val.(*redis.IntCmd).Result()
		total += intVal
	}

	return total, nil
}

// func (connector *DbConnector) removeNotificationsForTest(ctx context.Context, pipe redis.Pipeliner, notifications []*moira.ScheduledNotification, name string) (int64, error) {
// 	if len(notifications) == 0 {
// 		return 0, nil
// 	}

// 	for _, notification := range notifications {
// 		notificationString, err := reply.GetNotificationBytes(*notification)
// 		if err != nil {
// 			return 0, err
// 		}
// 		if name == "notifier-3" {
// 			time.Sleep(600 * time.Millisecond)
// 		}
// 		pipe.ZRem(ctx, notifierNotificationsKey, notificationString)
// 	}
// 	response, err := pipe.Exec(ctx)

// 	if err != nil {
// 		return 0, fmt.Errorf("failed to remove notifier-notification: %s", err.Error())
// 	}

// 	total := int64(0)
// 	for _, val := range response {
// 		intVal, _ := val.(*redis.IntCmd).Result()
// 		total += intVal
// 	}

// 	return total, nil
// }

// GetDelayedTimeInSeconds returns the time, if the difference between notification
// creation and sending time is greater than this time, the notification will be considered delayed
func (connector *DbConnector) GetDelayedTimeInSeconds() int64 {
	return int64(connector.notification.DelayedTime.Seconds())
}

// filterNotificationsByDelay filters notifications into delayed and not delayed notifications
func filterNotificationsByDelay(notifications []*moira.ScheduledNotification, delayedTime int64) ([]*moira.ScheduledNotification, []*moira.ScheduledNotification) {
	delayedNotifications := make([]*moira.ScheduledNotification, 0, len(notifications))
	notDelayedNotifications := make([]*moira.ScheduledNotification, 0, len(notifications))

	for _, notification := range notifications {
		if notification.IsDelayed(delayedTime) {
			delayedNotifications = append(delayedNotifications, notification)
			continue
		}
		notDelayedNotifications = append(notDelayedNotifications, notification)
	}

	return delayedNotifications, notDelayedNotifications
}

// getNotificationsTriggerChecks returns notifications trigger checks, optimized for the case when there are many notifications at one trigger
func (connector *DbConnector) getNotificationsTriggerChecks(notifications []*moira.ScheduledNotification) ([]*moira.CheckData, error) {
	checkDataMap := make(map[string]*moira.CheckData, len(notifications))
	for _, notification := range notifications {
		if notification != nil {
			checkDataMap[notification.Trigger.ID] = nil
		}
	}

	triggerIDs := make([]string, 0, len(checkDataMap))
	for triggerID := range checkDataMap {
		triggerIDs = append(triggerIDs, triggerID)
	}

	triggersLastCheck, err := connector.getTriggersLastCheck(triggerIDs)
	if err != nil {
		return nil, err
	}

	for i, triggerID := range triggerIDs {
		checkDataMap[triggerID] = triggersLastCheck[i]
	}

	result := make([]*moira.CheckData, len(notifications))
	for i, notification := range notifications {
		result[i] = checkDataMap[notification.Trigger.ID]
	}

	return result, nil
}

// filterNotificationsByState filters notifications based on their state to the corresponding arrays
func (connector *DbConnector) filterNotificationsByState(notifications []*moira.ScheduledNotification) ([]*moira.ScheduledNotification, []*moira.ScheduledNotification, error) {
	validNotifications := make([]*moira.ScheduledNotification, 0, len(notifications))
	toRemoveNotifications := make([]*moira.ScheduledNotification, 0, len(notifications))

	triggerChecks, err := connector.getNotificationsTriggerChecks(notifications)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get notifications trigger checks: %w", err)
	}

	for i := range notifications {
		switch notifications[i].GetState(triggerChecks[i]) {
		case moira.ValidNotification:
			validNotifications = append(validNotifications, notifications[i])

		case moira.RemovedNotification:
			toRemoveNotifications = append(toRemoveNotifications, notifications[i])
		}
	}

	return validNotifications, toRemoveNotifications, nil
}

/*
handleNotifications filters notifications into delayed and not delayed,
then filters delayed notifications by notification state, then merges the 2 arrays
of not delayed and valid delayed notifications into a single sorted array

Returns valid notifications in sorted order by timestamp and notifications to remove
*/
func (connector *DbConnector) handleNotifications(notifications []*moira.ScheduledNotification) ([]*moira.ScheduledNotification, []*moira.ScheduledNotification, error) {
	if len(notifications) == 0 {
		return notifications, nil, nil
	}

	delayedNotifications, notDelayedNotifications := filterNotificationsByDelay(notifications, connector.GetDelayedTimeInSeconds())
	connector.logger.Debug().
		Interface("delayed_notifications", delayedNotifications).
		Int("delayed_notifications_count", len(delayedNotifications)).
		Msg("Delayed notifications")

	connector.logger.Debug().
		Interface("not_delayed_notifications", notDelayedNotifications).
		Int("not_delayed_notifications_count", len(notDelayedNotifications)).
		Msg("Not delayed notifications")

	if len(delayedNotifications) == 0 {
		return notDelayedNotifications, notDelayedNotifications, nil
	}

	validNotifications, toRemoveNotifications, err := connector.filterNotificationsByState(delayedNotifications)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to filter delayed notifications by state: %w", err)
	}

	validNotifications, err = moira.MergeToSorted[*moira.ScheduledNotification](validNotifications, notDelayedNotifications)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge valid and not delayed notifications into sorted array: %w", err)
	}

	connector.logger.Debug().
		Interface("valid_notifications", validNotifications).
		Int("valid_notifications_count", len(validNotifications)).
		Msg("Valid notifications")

	connector.logger.Debug().
		Interface("to_remove_notifications", toRemoveNotifications).
		Int("to_remove_notifications_count", len(toRemoveNotifications)).
		Msg("To remove notifications")

	toRemoveNotifications = append(toRemoveNotifications, validNotifications...)

	return validNotifications, toRemoveNotifications, nil
}

// FetchNotifications fetch notifications by given timestamp and delete it
func (connector *DbConnector) FetchNotifications(to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	if limit == 0 {
		return nil, fmt.Errorf("limit mustn't be 0")
	}

	// No limit
	if limit == notifier.NotificationsLimitUnlimited {
		return connector.fetchNotifications(to, nil)
	}

	count, err := connector.notificationsCount(to)
	if err != nil {
		return nil, err
	}

	// Hope count will be not greater then limit when we call fetchNotificationsNoLimit
	if limit > transactionHeuristicLimit && count < limit/2 {
		return connector.fetchNotifications(to, nil)
	}

	return connector.fetchNotifications(to, &limit)
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
func (connector *DbConnector) fetchNotifications(to int64, limit *int64) ([]*moira.ScheduledNotification, error) {
	// fetchNotificationsDo uses WATCH, so transaction may fail and will retry it
	// see https://redis.io/topics/transactions

	for i := 0; i < connector.notification.TransactionMaxRetries; i++ {
		res, err := connector.fetchNotificationsDo(to, limit)

		if err == nil {
			return res, nil
		}

		if !errors.Is(err, &transactionError{}) {
			return nil, err
		}

		connector.logger.Info().
			Int("transaction_retries", i+1).
			Msg("Transaction error. Retry")

		time.Sleep(connector.notification.TransactionTimeout)
	}

	return nil, fmt.Errorf("transaction tries limit exceeded")
}

// fetchNotificationsWithLimit reads and drops notifications from DB with limit
func (connector *DbConnector) FetchNotificationsForTest(wg *sync.WaitGroup, to int64, limit *int64, name string) ([]*moira.ScheduledNotification, error) {
	defer wg.Done()
	// fetchNotificationsDo uses WATCH, so transaction may fail and will retry it
	// see https://redis.io/topics/transactions

	for i := 0; i < connector.notification.TransactionMaxRetries; i++ {
		res, err := connector.fetchNotificationsDoForTest(to, limit, name)

		if err == nil {
			log.Printf("%v result: \n", name)
			for _, r := range res {
				log.Println(r.Timestamp)
			}
			return res, nil
		}

		if !errors.Is(err, &transactionError{}) {
			return nil, err
		}

		log.Printf("%v: transaction retries %v\n", name, i)
		connector.logger.Info().
			Int("transaction_retries", i+1).
			Msg("Transaction error. Retry")

		time.Sleep(connector.notification.TransactionTimeout)
	}

	return nil, fmt.Errorf("transaction tries limit exceeded")
}

// getNotificationsInTxWithLimit receives notifications from the database by a certain time
// sorted by timestamp in one transaction with or without limit, depending on whether limit is nil
func getNotificationsInTxWithLimit(ctx context.Context, tx *redis.Tx, to int64, limit *int64) ([]*moira.ScheduledNotification, error) {
	var rng *redis.ZRangeBy
	if limit != nil {
		rng = &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10), Offset: 0, Count: *limit}
	} else {
		rng = &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10)}
	}

	response := tx.ZRangeByScore(ctx, notifierNotificationsKey, rng)
	if response.Err() != nil {
		return nil, fmt.Errorf("failed to ZRANGEBYSCORE: %w", response.Err())
	}

	return reply.Notifications(response)
}

/*
getLimitedNotifications restricts the list of notifications by last timestamp. There are two possible cases
with arrays of notifications with timestamps:

  - [1, 2, 3, 3], after limitNotifications we get the array [1, 2],
    further, since the array size is not equal to the passed one, we return [1, 2]

  - [1, 1, 1], after limitNotifications we will get array [1, 1, 1], its size is equal to the initial one,
    so we will get all notifications from the database with the last timestamp <= 1, i.e.,
    if the database at this moment has [1, 1, 1, 1, 1], then the output will be [1, 1, 1, 1, 1]

This is to ensure that notifications with the same timestamp are always clumped into a single stack
*/
func getLimitedNotifications(
	ctx context.Context,
	tx *redis.Tx,
	limit *int64,
	notifications []*moira.ScheduledNotification,
) ([]*moira.ScheduledNotification, error) {
	if len(notifications) == 0 {
		return notifications, nil
	}

	limitedNotifications := notifications

	if limit != nil {
		limitedNotifications = limitNotifications(notifications)
		lastTs := limitedNotifications[len(limitedNotifications)-1].Timestamp

		if len(notifications) == len(limitedNotifications) {
			// this means that all notifications have same timestamp,
			// we hope that all notifications with same timestamp should fit our memory
			var err error
			limitedNotifications, err = getNotificationsInTxWithLimit(ctx, tx, lastTs, nil)
			if err != nil {
				if errors.Is(err, redis.TxFailedErr) {
					return nil, &transactionError{}
				}
				return nil, fmt.Errorf("failed to get notification without limit in transaction: %w", err)
			}
		}
	}

	return limitedNotifications, nil
}

// fetchNotificationsDo performs fetching of notifications within a single transaction
func (connector *DbConnector) fetchNotificationsDo(to int64, limit *int64) ([]*moira.ScheduledNotification, error) {
	// See https://redis.io/topics/transactions
	if limit != nil {
		connector.logger.Debug().
			Int64("to", to).
			Int64("limit", *limit).
			Msg("Fetch notifications with limit")
	} else {
		connector.logger.Debug().
			Int64("to", to).
			Msg("Fetch notifications without limit")
	}

	ctx := connector.context
	c := *connector.client

	result := make([]*moira.ScheduledNotification, 0)

	err := c.Watch(ctx, func(tx *redis.Tx) error {
		notifications, err := getNotificationsInTxWithLimit(ctx, tx, to, limit)
		if err != nil {
			if errors.Is(err, redis.TxFailedErr) {
				return &transactionError{}
			}
			return fmt.Errorf("failed to get notifications with limit in transaction: %w", err)
		}

		if len(notifications) == 0 {
			return nil
		}

		// ZRANGEBYSCORE with LIMIT may return not all notifications with last timestamp
		// (e.g. we have notifications with timestamps [1, 2, 3, 3, 3] and limit = 3,
		// we will get [1, 2, 3]) other notifications with timestamp 3 remain in the database, so then
		// limit notifications by last timestamp and return and delete valid notifications with our new timestamp [1, 2]
		limitedNotifications, err := getLimitedNotifications(ctx, tx, limit, notifications)
		if err != nil {
			return fmt.Errorf("failed to get limited notifications: %w", err)
		}

		validNotifications, toRemoveNotifications, err := connector.handleNotifications(limitedNotifications)
		if err != nil {
			return fmt.Errorf("failed to validate notifications: %w", err)
		}
		result = validNotifications

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// someone has changed notifierNotificationsKey while we do our job
			// and transaction fail (no notifications were deleted) :(
			if _, err = connector.removeNotifications(ctx, pipe, toRemoveNotifications); err != nil {
				if errors.Is(err, redis.TxFailedErr) {
					return &transactionError{}
				}
				return fmt.Errorf("failed to remove notifications in transaction: %w", err)
			}

			return nil
		})

		return err
	}, notifierNotificationsKey)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// fetchNotificationsDo performs fetching of notifications within a single transaction
func (connector *DbConnector) fetchNotificationsDoForTest(to int64, limit *int64, name string) ([]*moira.ScheduledNotification, error) {
	// See https://redis.io/topics/transactions
	if limit != nil {
		connector.logger.Debug().
			Int64("to", to).
			Int64("limit", *limit).
			Interface("name", name).
			Msg("Fetch notifications with limit")
		log.Printf("name: %v, to: %v, limit: %v, fetch notifications with limit\n", name, to, *limit)
	} else {
		connector.logger.Debug().
			Int64("to", to).
			Interface("name", name).
			Msg("Fetch notifications without limit")
		log.Printf("name: %v, to: %v, fetch notifications without limit\n", name, to)
	}

	ctx := connector.context
	c := *connector.client

	result := make([]*moira.ScheduledNotification, 0)

	err := c.Watch(ctx, func(tx *redis.Tx) error {
		notifications, err := getNotificationsInTxWithLimit(ctx, tx, to, limit)
		if err != nil {
			if err == redis.TxFailedErr {
				return &transactionError{}
			}
			return fmt.Errorf("failed to get notifications with limit in transaction: %w", err)
		}

		if len(notifications) == 0 {
			return nil
		}

		// ZRANGEBYSCORE with LIMIT may return not all notifications with last timestamp
		// (e.g. we have notifications with timestamps [1, 2, 3, 3, 3] and limit = 3,
		// we will get [1, 2, 3]) other notifications with timestamp 3 remain in the database, so then
		// limit notifications by last timestamp and return and delete valid notifications with our new timestamp [1, 2]
		limitedNotifications, err := getLimitedNotifications(ctx, tx, limit, notifications)
		if err != nil {
			return fmt.Errorf("failed to get limited notifications: %w", err)
		}
		log.Printf("name: %v, limited notifications: \n", name)

		validNotifications, toRemoveNotifications, err := connector.handleNotifications(limitedNotifications)
		if err != nil {
			return fmt.Errorf("failed to validate notifications: %w", err)
		}
		result = validNotifications

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// someone has changed notifierNotificationsKey while we do our job
			// and transaction fail (no notifications were deleted) :(
			if _, err = connector.removeNotifications(ctx, pipe, toRemoveNotifications); err != nil {
				if errors.Is(err, redis.TxFailedErr) {
					return &transactionError{}
				}
				return fmt.Errorf("failed to remove notifications in transaction: %w", err)
			}

			return nil
		})

		return err
	}, notifierNotificationsKey)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// // same as fetchNotifications, but only once
// func (connector *DbConnector) fetchNotificationsDoForTest(to int64, limit *int64, name string) ([]*moira.ScheduledNotification, error) {
// 	// see https://redis.io/topics/transactions
// 	if limit != nil {
// 		connector.logger.Debug().
// 			Int64("to", to).
// 			Int64("limit", *limit).
// 			Interface("name", name).
// 			Msg("Fetch notifications with limit")
// 		log.Printf("name: %v, to: %v, limit: %v, fetch notifications with limit\n", name, to, *limit)
// 	} else {
// 		connector.logger.Debug().
// 			Int64("to", to).
// 			Interface("name", name).
// 			Msg("Fetch notifications without limit")
// 		log.Printf("name: %v, to: %v, fetch notifications without limit\n", name, to)
// 	}

// 	ctx := connector.context
// 	c := *connector.client

// 	result := make([]*moira.ScheduledNotification, 0)

// 	err := c.Watch(ctx, func(tx *redis.Tx) error {
// 		notifications, err := getNotificationsInTxWithLimit(ctx, tx, to, limit)
// 		if err != nil {
// 			return fmt.Errorf("failed to get notifications with limit in transaction: %w", err)
// 		}

// 		if len(notifications) == 0 {
// 			return nil
// 		}

// 		// ZRANGEBYSCORE with LIMIT may return not all notifications with last timestamp
// 		// (e.g. we have notifications with timestamps [1, 2, 3, 3, 3] and limit = 3,
// 		// we will get [1, 2, 3]) other notifications with timestamp 3 remain in the database, so then
// 		// limit notifications by last timestamp and return and delete valid notifications with our new timestamp [1, 2]
// 		limitedNotifications, err := getLimitedNotifications(ctx, tx, limit, notifications)
// 		if err != nil {
// 			return fmt.Errorf("failed to get limited notifications: %w", err)
// 		}
// 		log.Printf("name: %v, limited notifications: \n", name)
// 		for _, v := range limitedNotifications {
// 			log.Printf("name: %v, value: %v\n", name, v.Timestamp)
// 		}

// 		validNotifications, toRemoveNotifications, err := connector.handleNotifications(limitedNotifications)
// 		if err != nil {
// 			return fmt.Errorf("failed to validate notifications: %w", err)
// 		}
// 		result = validNotifications

// 		// if name == "notifier-1" || name == "notifier-2" {
// 		// 	time.Sleep(150 * time.Millisecond)
// 		// }
// 		//time.Sleep(150 * time.Millisecond)

// 		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
// 			var deleteCount int64
// 			deleteCount, err = connector.removeNotificationsForTest(ctx, pipe, toRemoveNotifications, name)

// 			// someone has changed notifierNotificationsKey while we do our job
// 			// and transaction fail (no notifications were deleted) :(
// 			if err != nil || deleteCount == 0 {
// 				return &transactionError{}
// 			}
// 			return nil
// 		})

// 		return err
// 	}, notifierNotificationsKey)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return result, nil
// }

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
