package reply

import (
	"github.com/moira-alert/moira-alert"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

func Notification(rep interface{}, err error) (*moira.ScheduledNotification, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	notification := &moira.ScheduledNotification{}
	err = json.Unmarshal(bytes, notification)
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func Notifications(rep interface{}, err error) ([]*moira.ScheduledNotification, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
	}
	notifications := make([]*moira.ScheduledNotification, len(values))
	for i, kk := range values {
		notification, err2 := Notification(kk, err)
		if err2 != nil {
			return nil, err2
		}
		notifications[i] = notification
	}
	return notifications, nil
}
