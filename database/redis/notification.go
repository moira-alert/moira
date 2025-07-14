package redis

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// Separate const to prevent cyclic dependencies.
// Original const is declared in notifier package, notifier depends on all metric source packages.
// Thus it prevents us from using database in tests for local metric source.
const notificationsLimitUnlimited = int64(-1)

type notificationTypes struct {
	Valid, ToRemove, ToResaveNew, ToResaveOld []*moira.ScheduledNotification
}

// Drops all notifications with last timestamp.
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

// GetNotifications gets ScheduledNotifications in given range and full range, where 'start' and 'end' are indices of notifications.
func (connector *DbConnector) GetNotifications(start, end int64) ([]*moira.ScheduledNotification, int64, error) {
	return connector.getNotificationsBy(func(pipe redis.Pipeliner, redisKey string) {
		pipe.ZRange(connector.context, redisKey, start, end)
	})
}

func (connector *DbConnector) getNotificationsByTime(timeStart, timeEnd int64) ([]*moira.ScheduledNotification, int64, error) {
	var stop string
	if timeEnd < 0 {
		stop = "inf"
	} else {
		stop = strconv.FormatInt(timeEnd, 10)
	}

	rng := &redis.ZRangeBy{
		Min: strconv.FormatInt(timeStart, 10),
		Max: stop,
	}

	return connector.getNotificationsBy(func(pipe redis.Pipeliner, redisKey string) {
		pipe.ZRangeByScore(connector.context, redisKey, rng)
	})
}

func (connector *DbConnector) getNotificationsBy(query func(pipe redis.Pipeliner, redisKey string)) ([]*moira.ScheduledNotification, int64, error) {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	for _, clusterKey := range connector.clusterList {
		redisKey := makeNotifierNotificationsKey(clusterKey)
		query(pipe, redisKey)
		pipe.ZCard(ctx, redisKey)
	}

	response, err := pipe.Exec(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	if len(response) == 0 {
		return make([]*moira.ScheduledNotification, 0), 0, nil
	}

	total := int64(0)
	notifications := make([]*moira.ScheduledNotification, 0)

	for i := range connector.clusterList {
		t, err := response[2*i+1].(*redis.IntCmd).Result()
		if err != nil {
			return nil, 0, err
		}

		ns, err := reply.Notifications(response[2*i].(*redis.StringSliceCmd))
		if err != nil {
			return nil, 0, err
		}

		notifications = append(notifications, ns...)

		total += t
	}

	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].Timestamp < notifications[j].Timestamp
	})

	return notifications, total, nil
}

// RemoveAllNotifications delete all notifications.
func (connector *DbConnector) RemoveAllNotifications() error {
	ctx := connector.context
	c := *connector.client

	redisKeys := make([]string, 0, len(connector.clusterList))
	for _, clusterKey := range connector.clusterList {
		redisKeys = append(redisKeys, makeNotifierNotificationsKey(clusterKey))
	}

	if _, err := c.Del(ctx, redisKeys...).Result(); err != nil {
		return fmt.Errorf("failed to remove notification keys: %s", err.Error())
	}

	return nil
}

// RemoveNotification delete notifications by key = timestamp + contactID + subID.
func (connector *DbConnector) RemoveNotification(notificationId string) (int64, error) {
	countTotal := int64(0)

	for _, clusterKey := range connector.clusterList {
		count, err := connector.removeNotificationInKey(makeNotifierNotificationsKey(clusterKey), notificationId)
		countTotal += count

		if err != nil {
			return countTotal, err
		}
	}

	return countTotal, nil
}

func (connector *DbConnector) removeNotificationInKey(redisKey string, notificationId string) (int64, error) {
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
		if idstr == notificationId {
			foundNotifications = append(foundNotifications, notification)
		}
	}

	return connector.removeNotifications(redisKey, connector.context, (*connector.client).TxPipeline(), foundNotifications)
}

// RemoveFilteredNotifications deletes notifications ine time range from startTime to endTime,
// excluding the ones that have tag from ignoredTags.
func (connector *DbConnector) RemoveFilteredNotifications(start, end int64, ignoredTags []string) (int64, error) {
	countTotal := int64(0)

	for _, clusterKey := range connector.clusterList {
		count, err := connector.removeFilteredNotificationsInRedisKey(makeNotifierNotificationsKey(clusterKey), start, end, ignoredTags)
		countTotal += count

		if err != nil {
			return countTotal, err
		}
	}

	return countTotal, nil
}

func (connector *DbConnector) removeFilteredNotificationsInRedisKey(redisKey string, start, end int64, ignoredTags []string) (int64, error) {
	notifications, _, err := connector.getNotificationsByTime(start, end)
	if err != nil {
		return 0, err
	}

	foundNotifications := make([]*moira.ScheduledNotification, 0)

	ignoredTagsSet := make(map[string]struct{}, len(ignoredTags))
	for _, tag := range ignoredTags {
		ignoredTagsSet[tag] = struct{}{}
	}

outer:
	for _, notification := range notifications {
		for _, tag := range notification.Trigger.Tags {
			if _, ok := ignoredTagsSet[tag]; ok {
				continue outer
			}
		}

		foundNotifications = append(foundNotifications, notification)
	}

	return connector.removeNotifications(redisKey, connector.context, (*connector.client).TxPipeline(), foundNotifications)
}

func (connector *DbConnector) removeNotifications(
	redisKey string,
	ctx context.Context,
	pipe redis.Pipeliner,
	notifications []*moira.ScheduledNotification,
) (int64, error) {
	if len(notifications) == 0 {
		return 0, nil
	}

	for _, notification := range notifications {
		notificationString, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return 0, err
		}

		pipe.ZRem(ctx, redisKey, notificationString)
	}

	response, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to remove notifications: %w", err)
	}

	var total int64

	for _, val := range response {
		intVal, _ := val.(*redis.IntCmd).Result()
		total += intVal
	}

	return total, nil
}

// getDelayedTimeInSeconds returns the time, if the difference between notification
// creation and sending time is greater than this time, the notification will be considered delayed.
func (connector *DbConnector) getDelayedTimeInSeconds() int64 {
	return int64(connector.notification.DelayedTime.Seconds())
}

// filterNotificationsByDelay filters notifications into delayed and not delayed notifications.
func filterNotificationsByDelay(notifications []*moira.ScheduledNotification, delayedTime int64) (delayedNotifications []*moira.ScheduledNotification, notDelayedNotifications []*moira.ScheduledNotification) {
	delayedNotifications = make([]*moira.ScheduledNotification, 0, len(notifications))
	notDelayedNotifications = make([]*moira.ScheduledNotification, 0, len(notifications))

	for _, notification := range notifications {
		if notification == nil {
			continue
		}

		if notification.IsDelayed(delayedTime) {
			delayedNotifications = append(delayedNotifications, notification)
		} else {
			notDelayedNotifications = append(notDelayedNotifications, notification)
		}
	}

	return delayedNotifications, notDelayedNotifications
}

// getNotificationsTriggerChecks returns notifications trigger checks, optimized for the case when there are many notifications at one trigger.
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

	result := make([]*moira.CheckData, 0, len(notifications))
	for _, notification := range notifications {
		result = append(result, checkDataMap[notification.Trigger.ID])
	}

	return result, nil
}

// Helper function for logging information on to remove notifications.
func logToRemoveNotifications(logger moira.Logger, toRemoveNotifications []*moira.ScheduledNotification) {
	if len(toRemoveNotifications) == 0 {
		return
	}

	triggerIDsSet := make(map[string]struct{}, len(toRemoveNotifications))

	for _, removedNotification := range toRemoveNotifications {
		if removedNotification == nil {
			continue
		}

		if _, ok := triggerIDsSet[removedNotification.Trigger.ID]; !ok {
			triggerIDsSet[removedNotification.Trigger.ID] = struct{}{}
		}
	}

	triggerIDs := make([]string, 0, len(triggerIDsSet))
	for triggerID := range triggerIDsSet {
		triggerIDs = append(triggerIDs, triggerID)
	}

	logger.Info().
		Interface("notification_trigger_ids", triggerIDs).
		Int("to_remove_count", len(toRemoveNotifications)).
		Msg("To remove notifications")
}

// filterNotificationsByState filters notifications based on their state to the corresponding arrays.
func (connector *DbConnector) filterNotificationsByState(notifications []*moira.ScheduledNotification) (notificationTypes, error) {
	types := notificationTypes{
		Valid:       make([]*moira.ScheduledNotification, 0, len(notifications)),
		ToRemove:    make([]*moira.ScheduledNotification, 0, len(notifications)),
		ToResaveNew: make([]*moira.ScheduledNotification, 0, len(notifications)),
		ToResaveOld: make([]*moira.ScheduledNotification, 0, len(notifications)),
	}

	triggerChecks, err := connector.getNotificationsTriggerChecks(notifications)
	if err != nil {
		return notificationTypes{}, fmt.Errorf("failed to get notifications trigger checks: %w", err)
	}

	for i, notification := range notifications {
		if notification != nil {
			switch notification.GetState(triggerChecks[i]) {
			case moira.ValidNotification:
				types.Valid = append(types.Valid, notification)

			case moira.RemovedNotification:
				types.ToRemove = append(types.ToRemove, notification)

			case moira.ResavedNotification:
				types.ToResaveOld = append(types.ToResaveOld, notification)

				updatedNotification := *notification
				updatedNotification.Timestamp = time.Now().Add(connector.notification.ResaveTime).Unix()
				types.ToResaveNew = append(types.ToResaveNew, &updatedNotification)
			}
		}
	}

	logToRemoveNotifications(connector.logger, types.ToRemove)

	return types, nil
}

/*
handleNotifications filters notifications into delayed and not delayed,
then filters delayed notifications by notification state, then merges the 2 arrays
of not delayed and valid delayed notifications into a single sorted array

Returns valid notifications in sorted order by timestamp and notifications to remove.
*/
func (connector *DbConnector) handleNotifications(notifications []*moira.ScheduledNotification) (notificationTypes, error) {
	if len(notifications) == 0 {
		return notificationTypes{}, nil
	}

	delayedNotifications, notDelayedNotifications := filterNotificationsByDelay(notifications, connector.getDelayedTimeInSeconds())

	if len(delayedNotifications) == 0 {
		return notificationTypes{
			Valid:    notDelayedNotifications,
			ToRemove: notDelayedNotifications,
		}, nil
	}

	types, err := connector.filterNotificationsByState(delayedNotifications)
	if err != nil {
		return notificationTypes{}, fmt.Errorf("failed to filter delayed notifications by state: %w", err)
	}

	types.Valid, err = moira.MergeToSorted[*moira.ScheduledNotification](types.Valid, notDelayedNotifications)
	if err != nil {
		return notificationTypes{}, fmt.Errorf("failed to merge valid and not delayed notifications into sorted array: %w", err)
	}

	types.ToRemove = append(types.ToRemove, types.Valid...)

	return types, nil
}

// FetchNotifications fetch notifications by given timestamp and delete it.
func (connector *DbConnector) FetchNotifications(clusterKey moira.ClusterKey, to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	if limit == 0 {
		return nil, fmt.Errorf("limit mustn't be 0")
	}

	redisKey := makeNotifierNotificationsKey(clusterKey)

	// No limit
	if limit == notificationsLimitUnlimited {
		return connector.fetchNotifications(redisKey, to, notificationsLimitUnlimited)
	}

	count, err := connector.notificationsCount(redisKey, to)
	if err != nil {
		return nil, err
	}

	// Hope count will be not greater then limit when we call fetchNotificationsNoLimit
	if limit > connector.notification.TransactionHeuristicLimit && count < limit/2 {
		return connector.fetchNotifications(redisKey, to, notificationsLimitUnlimited)
	}

	return connector.fetchNotifications(redisKey, to, limit)
}

func (connector *DbConnector) notificationsCount(redisKey string, to int64) (int64, error) {
	ctx := connector.context
	c := *connector.client

	count, err := c.ZCount(ctx, redisKey, "-inf", strconv.FormatInt(to, 10)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to ZCOUNT to notificationsCount: %w", err)
	}

	return count, nil
}

// fetchNotificationsWithLimit reads and drops notifications from DB with limit.
func (connector *DbConnector) fetchNotifications(redisKey string, to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	// fetchNotificationsDo uses WATCH, so transaction may fail and will retry it
	// see https://redis.io/topics/transactions
	for i := range connector.notification.TransactionMaxRetries {
		res, err := connector.fetchNotificationsTx(redisKey, to, limit)

		if err == nil {
			return res, nil
		}

		if !errors.Is(err, redis.TxFailedErr) {
			return nil, err
		}

		connector.logger.Info().
			Error(err).
			Int("transaction_retries", i+1).
			Msg("Transaction error. Retry")

		time.Sleep(connector.notification.TransactionTimeout)
	}

	return nil, fmt.Errorf("transaction tries limit exceeded")
}

// getNotificationsInTxWithLimit receives notifications from the database by a certain time
// sorted by timestamp in one transaction with or without limit, depending on whether limit is nil.
func getNotificationsInTxWithLimit(redisKey string, ctx context.Context, tx *redis.Tx, to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	var rng *redis.ZRangeBy
	if limit != notificationsLimitUnlimited {
		rng = &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10), Offset: 0, Count: limit}
	} else {
		rng = &redis.ZRangeBy{Min: "-inf", Max: strconv.FormatInt(to, 10)}
	}

	response := tx.ZRangeByScore(ctx, redisKey, rng)
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

This is to ensure that notifications with the same timestamp are always clumped into a single stack.
*/
func getLimitedNotifications(
	redisKey string,
	ctx context.Context,
	tx *redis.Tx,
	limit int64,
	notifications []*moira.ScheduledNotification,
) ([]*moira.ScheduledNotification, error) {
	if len(notifications) == 0 {
		return notifications, nil
	}

	limitedNotifications := notifications

	if limit != notificationsLimitUnlimited {
		limitedNotifications = limitNotifications(notifications)
		lastTs := limitedNotifications[len(limitedNotifications)-1].Timestamp

		if len(notifications) == len(limitedNotifications) {
			// this means that all notifications have same timestamp,
			// we hope that all notifications with same timestamp should fit our memory
			var err error

			limitedNotifications, err = getNotificationsInTxWithLimit(redisKey, ctx, tx, lastTs, notificationsLimitUnlimited)
			if err != nil {
				return nil, fmt.Errorf("failed to get notification without limit in transaction: %w", err)
			}
		}
	}

	return limitedNotifications, nil
}

// fetchNotificationsTx performs fetching of notifications within a single transaction.
func (connector *DbConnector) fetchNotificationsTx(redisKey string, to int64, limit int64) ([]*moira.ScheduledNotification, error) {
	// See https://redis.io/topics/transactions
	ctx := connector.context
	c := *connector.client

	result := make([]*moira.ScheduledNotification, 0)

	// it is necessary to do these operations in one transaction to avoid data race
	err := c.Watch(ctx, func(tx *redis.Tx) error {
		notifications, err := getNotificationsInTxWithLimit(redisKey, ctx, tx, to, limit)
		if err != nil {
			return fmt.Errorf("failed to get notifications with limit in transaction: %w", err)
		}

		if len(notifications) == 0 {
			return nil
		}

		// ZRANGEBYSCORE with LIMIT may return not all notifications with last timestamp
		// (e.g. we have notifications with timestamps [1, 2, 3, 3, 3] and limit = 3,
		// we will get [1, 2, 3]) other notifications with timestamp 3 remain in the database, so then
		// limit notifications by last timestamp and return and delete valid notifications with our new timestamp [1, 2]
		limitedNotifications, err := getLimitedNotifications(redisKey, ctx, tx, limit, notifications)
		if err != nil {
			return fmt.Errorf("failed to get limited notifications: %w", err)
		}

		types, err := connector.handleNotifications(limitedNotifications)
		if err != nil {
			return fmt.Errorf("failed to handle notifications: %w", err)
		}

		result = types.Valid

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// someone has changed redisKey while we do our job
			// and transaction fail (no notifications were deleted) :(
			var deleted int64

			deleted, err = connector.removeNotifications(redisKey, ctx, pipe, types.ToRemove)
			if err != nil {
				return fmt.Errorf("failed to remove notifications in transaction: %w", err)
			}

			if int(deleted) != len(types.ToRemove) {
				return fmt.Errorf("number of deleted: %d does not match the number of notifications to be deleted: %d", int(deleted), len(types.ToRemove))
			}

			var affected int

			affected, err = connector.resaveNotifications(redisKey, ctx, pipe, types.ToResaveOld, types.ToResaveNew)
			if err != nil {
				return fmt.Errorf("failed to resave notifications in transaction: %w", err)
			}

			if affected != len(types.ToResaveOld)+len(types.ToResaveNew) {
				return fmt.Errorf("number of affected: %d does not match the number of notifications to be affected: %d", affected, len(types.ToResaveOld)+len(types.ToResaveNew))
			}

			return nil
		})

		return err
	}, redisKey)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AddNotification store notification at given timestamp.
func (connector *DbConnector) AddNotification(notification *moira.ScheduledNotification) error {
	bytes, err := reply.GetNotificationBytes(*notification)
	if err != nil {
		return err
	}

	ctx := connector.context
	c := *connector.client

	z := &redis.Z{Score: float64(notification.Timestamp), Member: bytes}

	redisKey := makeNotifierNotificationsKey(moira.MakeClusterKey(notification.Trigger.TriggerSource, notification.Trigger.ClusterId))

	_, err = c.ZAdd(ctx, redisKey, z).Result()
	if err != nil {
		return fmt.Errorf("failed to add scheduled notification: %s, error: %s", string(bytes), err.Error())
	}

	return err
}

// AddNotifications store notification at given timestamp.
func (connector *DbConnector) AddNotifications(notifications []*moira.ScheduledNotification, timestamp int64) error {
	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	for _, notification := range notifications {
		bytes, err := reply.GetNotificationBytes(*notification)
		if err != nil {
			return err
		}

		z := &redis.Z{Score: float64(timestamp), Member: bytes}
		redisKey := makeNotifierNotificationsKey(moira.MakeClusterKey(notification.Trigger.TriggerSource, notification.Trigger.ClusterId))
		pipe.ZAdd(ctx, redisKey, z)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

func (connector *DbConnector) resaveNotifications(
	redisKey string,
	ctx context.Context,
	pipe redis.Pipeliner,
	oldNotifications []*moira.ScheduledNotification,
	newNotifications []*moira.ScheduledNotification,
) (int, error) {
	for _, notification := range oldNotifications {
		if notification != nil {
			notificationString, err := reply.GetNotificationBytes(*notification)
			if err != nil {
				return 0, err
			}

			pipe.ZRem(ctx, redisKey, notificationString)
		}
	}

	for _, notification := range newNotifications {
		if notification != nil {
			notificationString, err := reply.GetNotificationBytes(*notification)
			if err != nil {
				return 0, err
			}

			z := &redis.Z{Score: float64(notification.Timestamp), Member: notificationString}
			pipe.ZAdd(ctx, redisKey, z)
		}
	}

	response, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to EXEC: %w", err)
	}

	var total int

	for _, val := range response {
		intVal, err := val.(*redis.IntCmd).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get result of intCmd response value: %w", err)
		}

		total += int(intVal)
	}

	return total, nil
}

func makeNotifierNotificationsKey(clusterKey moira.ClusterKey) string {
	if clusterKey.ClusterId == moira.ClusterNotSet {
		clusterKey.ClusterId = moira.DefaultCluster
	}

	if clusterKey.TriggerSource == moira.TriggerSourceNotSet {
		clusterKey.TriggerSource = moira.GraphiteLocal
	}

	if clusterKey == moira.DefaultLocalCluster {
		return notifierNotificationsKey
	}

	return fmt.Sprintf("%s:%s.%s", notifierNotificationsKey, clusterKey.TriggerSource, clusterKey.ClusterId)
}

var notifierNotificationsKey = "moira-notifier-notifications"
