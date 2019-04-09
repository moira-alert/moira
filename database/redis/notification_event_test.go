package redis

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

func TestNotificationEvents(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Notification events manipulation", t, func(c C) {
		Convey("Test push-get-get count-fetch", t, func(c C) {
			Convey("Should no events", t, func(c C) {
				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				c.So(total, ShouldEqual, 0)

				actual1, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual1, ShouldResemble, moira.NotificationEvent{})
			})

			Convey("Should has one events after push", t, func(c C) {
				err := dataBase.PushNotificationEvent(&notificationEvent, true)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent})

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				c.So(total, ShouldEqual, 1)

				actual1, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldResemble, notificationEvent)
			})

			Convey("Should has event by triggerID after fetch", t, func(c C) {
				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent})

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				c.So(total, ShouldEqual, 1)
			})

			Convey("Should no events to fetch after fetch", t, func(c C) {
				actual1, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual1, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test push-fetch multiple event by differ triggerIDs", t, func(c C) {
			Convey("Push events and get it by triggerIDs", t, func(c C) {
				err := dataBase.PushNotificationEvent(&notificationEvent1, true)
				c.So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent2, true)
				c.So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(notificationEvent1.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent1})

				total := dataBase.GetNotificationEventCount(notificationEvent1.TriggerID, 0)
				c.So(total, ShouldEqual, 1)

				actual, err = dataBase.GetNotificationEvents(notificationEvent2.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent2})

				total = dataBase.GetNotificationEventCount(notificationEvent2.TriggerID, 0)
				c.So(total, ShouldEqual, 1)
			})

			Convey("Fetch one of them and check for existing again", t, func(c C) {
				actual1, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeNil)
				c.So(actual1, ShouldResemble, notificationEvent1)

				actual, err := dataBase.GetNotificationEvents(notificationEvent1.TriggerID, 0, 1)
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent1})

				total := dataBase.GetNotificationEventCount(notificationEvent1.TriggerID, 0)
				c.So(total, ShouldEqual, 1)
			})

			Convey("Fetch second then fetch and and check for ErrNil", t, func(c C) {
				actual, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, notificationEvent2)

				actual, err = dataBase.FetchNotificationEvent()
				c.So(err, ShouldBeError)
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test get by ranges", t, func(c C) {
			now := time.Now().Unix()
			event := moira.NotificationEvent{
				Timestamp: now,
				State:     moira.StateNODATA,
				OldState:  moira.StateNODATA,
				TriggerID: uuid.Must(uuid.NewV4()).String(),
				Metric:    "my.metric",
			}

			err := dataBase.PushNotificationEvent(&event, true)
			c.So(err, ShouldBeNil)

			actual, err := dataBase.GetNotificationEvents(event.TriggerID, 0, 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []*moira.NotificationEvent{&event})

			total := dataBase.GetNotificationEventCount(event.TriggerID, 0)
			c.So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now-1)
			c.So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now)
			c.So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now+1)
			c.So(total, ShouldEqual, 0)

			actual, err = dataBase.GetNotificationEvents(event.TriggerID, 1, 1)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))
		})

		Convey("Test removing notification events", t, func(c C) {
			Convey("Should remove all notifications", t, func(c C) {
				err := dataBase.PushNotificationEvent(&notificationEvent, true)
				c.So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent1, true)
				c.So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent2, true)
				c.So(err, ShouldBeNil)

				err = dataBase.RemoveAllNotificationEvents()
				c.So(err, ShouldBeNil)

				actual, err := dataBase.FetchNotificationEvent()
				c.So(err, ShouldResemble, database.ErrNil)
				c.So(actual, ShouldResemble, moira.NotificationEvent{})
			})
		})
	})
}

func TestNotificationEventErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()

	var notificationEvent = moira.NotificationEvent{
		Timestamp: time.Now().Unix(),
		State:     moira.StateNODATA,
		OldState:  moira.StateNODATA,
		TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
		Metric:    "my.metric",
	}

	Convey("Should throw error when no connection", t, func(c C) {
		actual1, err := dataBase.GetNotificationEvents("123", 0, 1)
		c.So(actual1, ShouldBeNil)
		c.So(err, ShouldNotBeNil)

		err = dataBase.PushNotificationEvent(&notificationEvent, true)
		c.So(err, ShouldNotBeNil)

		total := dataBase.GetNotificationEventCount("123", 0)
		c.So(total, ShouldEqual, 0)

		actual2, err := dataBase.FetchNotificationEvent()
		c.So(actual2, ShouldResemble, moira.NotificationEvent{})
		c.So(err, ShouldNotBeNil)
	})
}

var notificationEvent = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     moira.StateNODATA,
	OldState:  moira.StateNODATA,
	TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
	Metric:    "my.metric",
}

var notificationEvent1 = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     moira.StateEXCEPTION,
	OldState:  moira.StateNODATA,
	TriggerID: uuid.Must(uuid.NewV4()).String(),
	Metric:    "my.metric",
}
var notificationEvent2 = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     moira.StateOK,
	OldState:  moira.StateWARN,
	TriggerID: uuid.Must(uuid.NewV4()).String(),
	Metric:    "my.metric1",
}
