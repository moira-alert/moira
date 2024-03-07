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

func getNotificationBytes(notification *moira.NotificationEventHistoryItem) ([]byte, error) {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %s", err.Error())
	}
	return bytes, nil
}

func getNotificationStruct(notificationString string) (moira.NotificationEventHistoryItem, error) {
	var object moira.NotificationEventHistoryItem
	err := json.Unmarshal([]byte(notificationString), &object)
	if err != nil {
		return object, fmt.Errorf("failed to umarshall event: %s", err.Error())
	}
	return object, nil
}

func (connector *DbConnector) GetNotificationsByContactIdWithLimit(contactID string, from int64, to int64) ([]*moira.NotificationEventHistoryItem, error) {
	c := *connector.client
	var notifications []*moira.NotificationEventHistoryItem

	notificationStings, err := c.ZRangeByScore(connector.context, contactNotificationKey, &redis.ZRangeBy{
		Min:   strconv.FormatInt(from, 10),
		Max:   strconv.FormatInt(to, 10),
		Count: int64(connector.notificationHistory.NotificationHistoryQueryLimit),
	}).Result()

	if err != nil {
		return notifications, err
	}

	for _, notification := range notificationStings {
		notificationObj, err := getNotificationStruct(notification)

		if err != nil {
			return notifications, err
		}

		if notificationObj.ContactID == contactID {
			notifications = append(notifications, &notificationObj)
		}
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

	notificationBytes, serializationErr := getNotificationBytes(notificationItemToSave)

	if serializationErr != nil {
		return fmt.Errorf("failed to serialize notification to contact event history item: %s", serializationErr.Error())
	}

	to := int(time.Now().Unix() - int64(connector.notificationHistory.NotificationHistoryTTL.Seconds()))

	pipe := (*connector.client).TxPipeline()

	pipe.ZAdd(
		connector.context,
		contactNotificationKey,
		&redis.Z{
			Score:  float64(notification.Timestamp),
			Member: notificationBytes})

	pipe.ZRemRangeByScore(
		connector.context,
		contactNotificationKey,
		"-inf",
		strconv.Itoa(to),
	)

	_, err := pipe.Exec(connector.Context())

	if err != nil {
		return fmt.Errorf("failed to push contact event history item: %s", err.Error())
	}

	return nil
}
