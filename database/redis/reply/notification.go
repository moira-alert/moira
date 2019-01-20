package reply

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// Notification converts redis DB reply to moira.ScheduledNotification object
func Notification(rep interface{}, err error) (moira.ScheduledNotification, error) {
	notification := moira.ScheduledNotification{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return notification, database.ErrNil
		}
		return notification, fmt.Errorf("failed to read scheduledNotification: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &notification)
	if err != nil {
		return notification, fmt.Errorf("failed to parse notification json %s: %s", string(bytes), err.Error())
	}
	return notification, nil
}

// Notifications converts redis DB reply to moira.ScheduledNotification objects array
func Notifications(rep interface{}, err error) ([]*moira.ScheduledNotification, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.ScheduledNotification, 0), nil
		}
		return nil, fmt.Errorf("failed to read ScheduledNotifications: %s", err.Error())
	}
	notifications := make([]*moira.ScheduledNotification, len(values))
	for i, value := range values {
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
