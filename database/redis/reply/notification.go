package reply

import (
	"encoding/json"
	"errors"
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
	CreatedAt int64                   `json:"created_at,omitempty"`
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
		CreatedAt: notification.CreatedAt,
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
		CreatedAt: n.CreatedAt,
	}
}

// GetNotificationBytes is a function that takes moira.ScheduledNotification and turns it to bytes that will be saved in redis.
func GetNotificationBytes(notification moira.ScheduledNotification) ([]byte, error) {
	notificationSE := toScheduledNotificationStorageElement(notification)
	bytes, err := json.Marshal(notificationSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification: %w", err)
	}
	return bytes, nil
}

// unmarshalNotification converts JSON to moira.ScheduledNotification object
func unmarshalNotification(bytes []byte, err error) (moira.ScheduledNotification, error) {
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return moira.ScheduledNotification{}, database.ErrNil
		}
		return moira.ScheduledNotification{}, fmt.Errorf("failed to read scheduledNotification: %w", err)
	}

	notificationSE := scheduledNotificationStorageElement{}
	err = json.Unmarshal(bytes, &notificationSE)
	if err != nil {
		return moira.ScheduledNotification{}, fmt.Errorf("failed to parse notification json %s: %w", string(bytes), err)
	}

	return notificationSE.toScheduledNotification(), nil
}

// Notifications converts redis DB reply to moira.ScheduledNotification objects array
func Notifications(responses *redis.StringSliceCmd) ([]*moira.ScheduledNotification, error) {
	if responses == nil || errors.Is(responses.Err(), redis.Nil) {
		return make([]*moira.ScheduledNotification, 0), nil
	}

	data, err := responses.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read ScheduledNotifications: %w", err)
	}

	notifications := make([]*moira.ScheduledNotification, len(data))
	for i, value := range data {
		notification, err2 := unmarshalNotification([]byte(value), err)
		if err2 != nil && !errors.Is(err2, database.ErrNil) {
			return nil, err2
		}
		if errors.Is(err2, database.ErrNil) {
			notifications[i] = nil
		} else {
			notifications[i] = &notification
		}
	}

	return notifications, nil
}
