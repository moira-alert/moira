package reply

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// scheduledNotificationStorageElement represent notification object
type scheduledNotificationStorageElement struct {
	Event     moira.NotificationEvent `json:"event"`
	Trigger   moira.TriggerData       `json:"trigger"`
	Contact   moira.ContactData       `json:"contact"`
	Plotting  moira.PlottingData      `json:"plotting"`
	Throttled bool                    `json:"throttled"`
	SendFail  int                     `json:"send_fail"`
	Timestamp int64                   `json:"timestamp"`
}

func toScheduledNotificationStorageElement(notification moira.ScheduledNotification) scheduledNotificationStorageElement {
	return scheduledNotificationStorageElement{
		Event:     notification.Event,
		Trigger:   notification.Trigger,
		Contact:   notification.Contact,
		Plotting:  notification.Plotting,
		Throttled: notification.Throttled,
		SendFail:  notification.SendFail,
		Timestamp: notification.Timestamp,
	}
}

func (n scheduledNotificationStorageElement) toScheduledNotification() moira.ScheduledNotification {
	return moira.ScheduledNotification{
		Event:     n.Event,
		Trigger:   n.Trigger,
		Contact:   n.Contact,
		Plotting:  n.Plotting,
		Throttled: n.Throttled,
		SendFail:  n.SendFail,
		Timestamp: n.Timestamp,
	}
}

// GetNotificationBytes is a function that takes moira.ScheduledNotification and turns it to bytes that will be saved in redis.
func GetNotificationBytes(notification moira.ScheduledNotification) ([]byte, error) {
	notificationSE := toScheduledNotificationStorageElement(notification)
	bytes, err := json.Marshal(notificationSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification: %s", err.Error())
	}
	return bytes, nil
}

// Notification converts redis DB reply to moira.ScheduledNotification object
func Notification(response string, err error) (moira.ScheduledNotification, error) {
	bytes := []byte(response)
	if err != nil {
		if err == redis.Nil {
			return moira.ScheduledNotification{}, database.ErrNil
		}
		return moira.ScheduledNotification{}, fmt.Errorf("failed to read scheduledNotification: %s", err.Error())
	}
	notificationSE := scheduledNotificationStorageElement{}
	err = json.Unmarshal(bytes, &notificationSE)
	if err != nil {
		return moira.ScheduledNotification{}, fmt.Errorf("failed to parse notification json %s: %s", string(bytes), err.Error())
	}
	return notificationSE.toScheduledNotification(), nil
}

// Notifications converts redis DB reply to moira.ScheduledNotification objects array
func Notifications(responses []string, err error) ([]*moira.ScheduledNotification, error) {
	if err != nil {
		if err == redis.Nil {
			return make([]*moira.ScheduledNotification, 0), nil
		}
		return nil, fmt.Errorf("failed to read ScheduledNotifications: %s", err.Error())
	}

	notifications := make([]*moira.ScheduledNotification, len(responses))
	for i, value := range responses {
		notification, err2 := Notification(value, err)
		if err2 != nil && err2 != database.ErrNil {
			return nil, err2
		} else if err2 == database.ErrNil {
			notifications[i] = nil
		} else {
			notifications[i] = &notification
		}
	}
	return notifications, nil
}
