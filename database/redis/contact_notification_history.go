package redis

import (
	"encoding/json"
	"fmt"

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
		Min: from,
		Max: to, //offset and count here
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

func (connector *DbConnector) PushContactNotificationToHistory(notification *moira.ScheduledNotification) error {
	c := *connector.client

	notificationItemToSave := &moira.NotificationEventHistoryItem{
		Metric:    notification.Event.Metric,
		State:     notification.Event.State,
		TriggerID: notification.Trigger.ID,
		OldState:  notification.Event.OldState,
		ContactID: notification.Contact.ID,
		TimeStamp: notification.Timestamp,
	}

	notificationBytes, _ := getNotificationBytes(notificationItemToSave)

	_, err := c.ZAdd(
		connector.context,
		contactNotificationKey,
		&redis.Z{
			Score:  float64(notification.Timestamp),
			Member: notificationBytes}).Result()

	return err
}
