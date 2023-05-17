package redis

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"time"
)

type NotificationHistoryItem struct {
	NotificationTimestamp int64
	TriggerState          moira.State
	ContactID             string
	TriggerId             string
}

const contactNotificationKey string = "moira-contact-notifications"

func (connector *DbConnector) GetAllNotificationsByContactId(contactID string) ([]string, error) {
	c := *connector.client
	var notifications []string

	notificationStrings, _ := c.ZRange(connector.context, contactNotificationKey, 0, time.Now().Unix()).Result()

	for _, notification := range notificationStrings {
		var notificationObj NotificationHistoryItem
		err := json.Unmarshal([]byte(notification), &notificationObj)
		if err != nil {
			fmt.Printf("Error parsing notification from db")
		}

		if notificationObj.ContactID == contactID {
			notifications = append(notifications, notification)
		}
	}

	return notifications, nil
}

func (connector *DbConnector) PushContactNotificationToHistory(notification *moira.ScheduledNotification) error {
	c := *connector.client

	notificationItemToSave := &NotificationHistoryItem{
		NotificationTimestamp: notification.Timestamp,
		TriggerState:          notification.Event.State,
		ContactID:             notification.Contact.ID,
		TriggerId:             notification.Trigger.ID,
	}

	notificationBytes, _ := GetNotificationBytes(notificationItemToSave)

	_, err := c.ZAdd(
		connector.context,
		contactNotificationKey,
		&redis.Z{
			Score:  float64(notificationItemToSave.NotificationTimestamp),
			Member: notificationBytes}).Result()

	return err
}

func GetNotificationBytes(notification *NotificationHistoryItem) ([]byte, error) {
	bytes, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification event: %s", err.Error())
	}
	return bytes, nil
}
