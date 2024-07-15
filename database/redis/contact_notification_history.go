package redis

import (
	"encoding/json"
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

// GetNotificationStruct unmarshals moira.NotificationEventHistoryItem from json represented by sting.
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

// GetNotificationsHistoryByContactId returns `size` (or all if `size` is -1) notification events with timestamp between `from` and `to`.
// The offset for fetching events may be changed by using `page` parameter, it is calculated as page * size.
func (connector *DbConnector) GetNotificationsHistoryByContactId(contactID string, from, to, page, size int64,
) ([]*moira.NotificationEventHistoryItem, error) {
	c := *connector.client

	notificationStings, err := c.ZRangeByScore(
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

	notifications := make([]*moira.NotificationEventHistoryItem, 0, len(notificationStings))

	for _, notification := range notificationStings {
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
	toDeleteFrom := "-inf"
	toDeleteTo := strconv.Itoa(int(time.Now().Unix() - ttl))

	client := connector.Client()
	pipe := client.TxPipeline()

	iterator := client.Scan(connector.context, 0, contactNotificationKeyWithID("*"), 0).Iterator()
	for iterator.Next(connector.context) {
		pipe.ZRemRangeByScore(
			connector.context,
			iterator.Val(),
			toDeleteFrom,
			toDeleteTo,
		)
	}

	if err := iterator.Err(); err != nil {
		return err
	}

	_, err := pipe.Exec(connector.context)
	if err != nil {
		return err
	}

	return nil
}
