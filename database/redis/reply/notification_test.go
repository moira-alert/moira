package reply

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	. "github.com/smartystreets/goconvey/convey"
)

const expectedBytes = `{"event":{"timestamp":0,"metric":"","state":"","trigger_id":"","old_state":"","event_message":null},"trigger":{"id":"","name":"","desc":"","targets":null,"warn_value":0,"error_value":0,"is_remote":false,"__notifier_trigger_tags":null},"contact":{"type":"","value":"","id":"","user":"","team":""},"plotting":{"enabled":false,"theme":""},"throttled":false,"send_fail":0,"timestamp":0}`

func TestGetNotificationBytes(t *testing.T) {
	Convey("Test GetNotificationBytes", t, func() {
		Convey("Test without created_at", func() {
			notification := moira.ScheduledNotification{
				Event:     moira.NotificationEvent{},
				Trigger:   moira.TriggerData{},
				Contact:   moira.ContactData{},
				Plotting:  moira.PlottingData{},
				Throttled: false,
				SendFail:  0,
				Timestamp: 0,
			}

			bytes, err := GetNotificationBytes(notification)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldEqual, expectedBytes)
		})

		Convey("Test with zero created_at", func() {
			notification := moira.ScheduledNotification{
				Event:     moira.NotificationEvent{},
				Trigger:   moira.TriggerData{},
				Contact:   moira.ContactData{},
				Plotting:  moira.PlottingData{},
				Throttled: false,
				SendFail:  0,
				Timestamp: 0,
				CreatedAt: 0,
			}

			bytes, err := GetNotificationBytes(notification)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldEqual, expectedBytes)
		})
	})
}

func TestUnmarshalNotification(t *testing.T) {
	Convey("Test unmarshal notification", t, func() {
		Convey("Test without error", func() {
			notification, err := unmarshalNotification([]byte(expectedBytes), nil)
			So(err, ShouldBeNil)
			So(notification, ShouldResemble, moira.ScheduledNotification{
				Event:     moira.NotificationEvent{},
				Trigger:   moira.TriggerData{},
				Contact:   moira.ContactData{},
				Plotting:  moira.PlottingData{},
				Throttled: false,
				SendFail:  0,
				Timestamp: 0,
				CreatedAt: 0,
			})
		})

		Convey("Test with redis.Nil error", func() {
			notification, err := unmarshalNotification([]byte(expectedBytes), redis.Nil)
			So(err, ShouldResemble, database.ErrNil)
			So(notification, ShouldResemble, moira.ScheduledNotification{})
		})

		Convey("Test with some error", func() {
			testErr := errors.New("Test err")
			notification, err := unmarshalNotification([]byte(expectedBytes), testErr)
			So(err, ShouldResemble, fmt.Errorf("failed to read scheduledNotification: %w", testErr))
			So(notification, ShouldResemble, moira.ScheduledNotification{})
		})
	})
}

func TestNotifications(t *testing.T) {
	Convey("Test Notifications", t, func() {
		Convey("Test with nil responses", func() {
			notifications, err := Notifications(nil)
			So(err, ShouldBeNil)
			So(notifications, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test with notification", func() {
			responses := &redis.StringSliceCmd{}
			responses.SetVal([]string{expectedBytes})

			notifications, err := Notifications(responses)
			So(err, ShouldBeNil)
			So(notifications, ShouldResemble, []*moira.ScheduledNotification{
				{
					Event:     moira.NotificationEvent{},
					Trigger:   moira.TriggerData{},
					Contact:   moira.ContactData{},
					Plotting:  moira.PlottingData{},
					Throttled: false,
					SendFail:  0,
					Timestamp: 0,
					CreatedAt: 0,
				},
			})
		})
	})
}
