package redis

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"time"
)

const contactNotificationKey string = "moira-contact-notifications"

func (connector *DbConnector) GetAllNotificationsByContactId(contactID string) (moira.NotificationEvents, error) {
	c := *connector.client
	var notifications moira.NotificationEvents

	notificationStrings, _ := c.ZRange(connector.context, contactNotificationKey, 0, time.Now().Unix()).Result()

	for _, notification := range notificationStrings {
		notificationObj, err := GetNotificationStruct(notification)
		if err != nil {
			fmt.Printf("Error parsing notification from db")
		}

		if notificationObj.ContactID == contactID {
			notifications = append(notifications, notificationObj)
		}
	}

	return notifications, nil
}

func (connector *DbConnector) PushContactNotificationToHistory(notification *moira.ScheduledNotification) error {
	c := *connector.client

	notificationItemToSave := &moira.NotificationEvent{
		Timestamp: notification.Timestamp,
		Metric:    notification.Event.Metric,
		State:     notification.Event.State,
		TriggerID: notification.Trigger.ID,
		ContactID: notification.Contact.ID,
		OldState:  notification.Event.OldState,
	}

	notificationBytes, _ := GetNotificationBytes(notificationItemToSave)

	_, err := c.ZAdd(
		connector.context,
		contactNotificationKey,
		&redis.Z{
			Score:  float64(notificationItemToSave.Timestamp),
			Member: notificationBytes}).Result()

	return err
}

func GetNotificationBytes(notification *moira.NotificationEvent) ([]byte, error) {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %s", err.Error())
	}
	return bytes, nil
}

func GetNotificationStruct(notificationString string) (moira.NotificationEvent, error) {
	var object moira.NotificationEvent
	err := json.Unmarshal([]byte(notificationString), &object)
	if err != nil {
		return object, fmt.Errorf("failed to umarshell event: %s", err.Error())
	}
	return object, nil
}
