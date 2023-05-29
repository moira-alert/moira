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

func (connector *DbConnector) GetNotificationsByContactIdWithLimit(contactID string, from string, to string) ([]*moira.NotificationEventHistoryItem, error) {
	c := *connector.client
	var notifications []*moira.NotificationEventHistoryItem

	notificationStings, _ := c.ZRangeByScore(connector.context, contactNotificationKey, &redis.ZRangeBy{
		Min:   from,
		Max:   to,
		Count: connector.NotificationHistoryQueryLimit,
	}).Result()

	for _, notification := range notificationStings {
		notificationObj, err := getNotificationStruct(notification)
		if err != nil {
			fmt.Printf("Error parsing notification from db")
		}

		if notificationObj.ContactID == contactID {
			notifications = append(notifications, &notificationObj)
		}
	}

	return notifications, nil
}

// PushContactNotificationToHistory converts ScheduledNotification to NotificationEventHistoryItem and
// saves it, and deletes items older than specified ttl
func (connector *DbConnector) PushContactNotificationToHistory(notification *moira.ScheduledNotification) error {
	notificationItemToSave := &moira.NotificationEventHistoryItem{
		Metric:    notification.Event.Metric,
		State:     notification.Event.State,
		TriggerID: notification.Trigger.ID,
		OldState:  notification.Event.OldState,
		ContactID: notification.Contact.ID,
		TimeStamp: notification.Timestamp,
	}

	notificationBytes, _ := getNotificationBytes(notificationItemToSave)

	to := int(time.Now().Unix() - connector.NotificationHistoryTtlSeconds)

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
