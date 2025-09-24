package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

const contactNotificationKey = "moira-contact-notifications"

// GetNotificationBytes marshals moira.NotificationHistoryItem to json.
func GetNotificationBytes(notification *moira.NotificationEventHistoryItem) ([]byte, error) {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %w", err)
	}

	return bytes, nil
}

// GetNotificationStruct unmarshals moira.NotificationEventHistoryItem from json represented by string.
func GetNotificationStruct(notificationString string) (moira.NotificationEventHistoryItem, error) {
	var object moira.NotificationEventHistoryItem

	err := json.Unmarshal([]byte(notificationString), &object)
	if err != nil {
		return object, fmt.Errorf("failed to umarshal event: %w", err)
	}

	return object, nil
}

func contactNotificationKeyWithID(contactID string) string {
	return contactNotificationKey + ":" + contactID
}

// GetNotificationsTotalByContactID returns total count of notification events by contactId.
func (connector *DbConnector) GetNotificationsTotalByContactID(contactID string, from, to int64) (int64, error) {
	c := *connector.client

	total, err := c.ZCount(
		connector.context,
		contactNotificationKeyWithID(contactID),
		strconv.FormatInt(from, 10),
		strconv.FormatInt(to, 10),
	).Result()
	if err != nil {
		return 0, err
	}

	return total, nil
}

// GetNotificationsHistoryByContactID returns `size` (or all if `size` is -1) notification events with timestamp between `from` and `to`.
// The offset for fetching events may be changed by using `page` parameter, it is calculated as page * size.
func (connector *DbConnector) GetNotificationsHistoryByContactID(contactID string, from, to, page, size int64,
) ([]*moira.NotificationEventHistoryItem, error) {
	c := *connector.client

	notificationStrings, err := c.ZRangeByScore(
		connector.context,
		contactNotificationKeyWithID(contactID),
		&redis.ZRangeBy{
			Min:    strconv.FormatInt(from, 10),
			Max:    strconv.FormatInt(to, 10),
			Offset: page * size,
			Count:  size,
		}).Result()
	if err != nil {
		return nil, err
	}

	notifications := make([]*moira.NotificationEventHistoryItem, 0, len(notificationStrings))

	for _, notification := range notificationStrings {
		notificationObj, err := GetNotificationStruct(notification)
		if err != nil {
			return notifications, err
		}

		notifications = append(notifications, &notificationObj)
	}

	return notifications, nil
}

// PushContactNotificationToHistory converts ScheduledNotification to NotificationEventHistoryItem and saves it,
// and deletes items older than specified ttl.
func (connector *DbConnector) PushContactNotificationToHistory(notification *moira.ScheduledNotification) error {
	notificationItemToSave := &moira.NotificationEventHistoryItem{
		Metric:    notification.Event.Metric,
		State:     notification.Event.State,
		TriggerID: notification.Trigger.ID,
		OldState:  notification.Event.OldState,
		ContactID: notification.Contact.ID,
		TimeStamp: notification.Timestamp,
	}

	notificationBytes, serializationErr := GetNotificationBytes(notificationItemToSave)

	if serializationErr != nil {
		return fmt.Errorf("failed to serialize notification to contact event history item: %w", serializationErr)
	}

	to := int(time.Now().Unix() - int64(connector.notificationHistory.NotificationHistoryTTL.Seconds()))

	pipe := (*connector.client).TxPipeline()

	pipe.ZAdd(
		connector.context,
		contactNotificationKeyWithID(notificationItemToSave.ContactID),
		&redis.Z{
			Score:  float64(notification.Timestamp),
			Member: notificationBytes,
		})

	pipe.ZRemRangeByScore(
		connector.context,
		contactNotificationKeyWithID(notificationItemToSave.ContactID),
		"-inf",
		strconv.Itoa(to),
	)

	_, err := pipe.Exec(connector.Context())
	if err != nil {
		return fmt.Errorf("failed to push contact event history item: %w", err)
	}

	return nil
}

// CleanUpOutdatedNotificationHistory is used for deleting notification history events which have been created more than ttl ago.
func (connector *DbConnector) CleanUpOutdatedNotificationHistory(ttl int64) error {
	return connector.callFunc(func(dbConn *DbConnector, client redis.UniversalClient) error {
		from := "-inf"
		to := strconv.Itoa(int(time.Now().Unix() - ttl))

		ctx := dbConn.Context()

		cmds, err := client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			iterator := client.Scan(ctx, 0, contactNotificationKeyWithID("*"), 0).Iterator()
			for iterator.Next(ctx) {
				pipe.ZRemRangeByScore(
					ctx,
					iterator.Val(),
					from,
					to,
				)
			}

			if err := iterator.Err(); err != nil {
				return fmt.Errorf("failed to iterate over notification history keys: %w", err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to pipeline deleting: %w", err)
		}

		var totalDelCount int64

		for _, cmd := range cmds {
			count, err := cmd.(*redis.IntCmd).Result()
			if err != nil {
				connector.logger.Info().
					Error(err).
					Msg("failed to remove outdated")
			}

			totalDelCount += count
		}

		connector.logger.Info().
			Int64("delete_count", totalDelCount).
			Msg("Cleaned up notification history")

		return nil
	})
}

// CountEventsInNotificationHistory returns the number of events in time range (from, to) for given contact ids.
func (connector *DbConnector) CountEventsInNotificationHistory(contactIDs []string, from, to string) ([]*moira.ContactIDWithNotificationCount, error) {
	pipe := connector.Client().TxPipeline()
	ctx := connector.Context()

	for _, id := range contactIDs {
		pipe.ZCount(ctx, contactNotificationKeyWithID(id), from, to)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	eventsCount := make([]*moira.ContactIDWithNotificationCount, 0, len(cmds))

	for i, cmd := range cmds {
		count, err := cmd.(*redis.IntCmd).Uint64()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}

		eventsCount = append(eventsCount, &moira.ContactIDWithNotificationCount{
			ID:    contactIDs[i],
			Count: count,
		})
	}

	return eventsCount, nil
}
