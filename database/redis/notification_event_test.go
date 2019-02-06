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

	Convey("Notification events manipulation", t, func() {
		Convey("Test push-get-get count-fetch", func() {
			Convey("Should no events", func() {
				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				So(total, ShouldEqual, 0)

				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{})
			})

			Convey("Should has one events after push", func() {
				err := dataBase.PushNotificationEvent(&notificationEvent, true)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent})

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				So(total, ShouldEqual, 1)

				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, notificationEvent)
			})

			Convey("Should has event by triggerID after fetch", func() {
				actual, err := dataBase.GetNotificationEvents(notificationEvent.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent})

				total := dataBase.GetNotificationEventCount(notificationEvent.TriggerID, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Should no events to fetch after fetch", func() {
				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test push-fetch multiple event by differ triggerIDs", func() {
			Convey("Push events and get it by triggerIDs", func() {
				err := dataBase.PushNotificationEvent(&notificationEvent1, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent2, true)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(notificationEvent1.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent1})

				total := dataBase.GetNotificationEventCount(notificationEvent1.TriggerID, 0)
				So(total, ShouldEqual, 1)

				actual, err = dataBase.GetNotificationEvents(notificationEvent2.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent2})

				total = dataBase.GetNotificationEventCount(notificationEvent2.TriggerID, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Fetch one of them and check for existing again", func() {
				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, notificationEvent1)

				actual, err := dataBase.GetNotificationEvents(notificationEvent1.TriggerID, 0, 1)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{&notificationEvent1})

				total := dataBase.GetNotificationEventCount(notificationEvent1.TriggerID, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Fetch second then fetch and and check for ErrNil", func() {
				actual, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, notificationEvent2)

				actual, err = dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test get by ranges", func() {
			now := time.Now().Unix()
			event := moira.NotificationEvent{
				Timestamp: now,
				State:     "NODATA",
				OldState:  "NODATA",
				TriggerID: uuid.Must(uuid.NewV4()).String(),
				Metric:    "my.metric",
			}

			err := dataBase.PushNotificationEvent(&event, true)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetNotificationEvents(event.TriggerID, 0, 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.NotificationEvent{&event})

			total := dataBase.GetNotificationEventCount(event.TriggerID, 0)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now-1)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(event.TriggerID, now+1)
			So(total, ShouldEqual, 0)

			actual, err = dataBase.GetNotificationEvents(event.TriggerID, 1, 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))
		})

		Convey("Test removing notification events", func() {
			Convey("Should remove all notifications", func() {
				err := dataBase.PushNotificationEvent(&notificationEvent, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent1, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&notificationEvent2, true)
				So(err, ShouldBeNil)

				err = dataBase.RemoveAllNotificationEvents()
				So(err, ShouldBeNil)

				actual, err := dataBase.FetchNotificationEvent()
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.NotificationEvent{})
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
		State:     "NODATA",
		OldState:  "NODATA",
		TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
		Metric:    "my.metric",
	}

	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetNotificationEvents("123", 0, 1)
		So(actual1, ShouldBeNil)
		So(err, ShouldNotBeNil)

		err = dataBase.PushNotificationEvent(&notificationEvent, true)
		So(err, ShouldNotBeNil)

		total := dataBase.GetNotificationEventCount("123", 0)
		So(total, ShouldEqual, 0)

		actual2, err := dataBase.FetchNotificationEvent()
		So(actual2, ShouldResemble, moira.NotificationEvent{})
		So(err, ShouldNotBeNil)
	})
}

var notificationEvent = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     "NODATA",
	OldState:  "NODATA",
	TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
	Metric:    "my.metric",
}

var notificationEvent1 = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     "EXCEPTION",
	OldState:  "NODATA",
	TriggerID: uuid.Must(uuid.NewV4()).String(),
	Metric:    "my.metric",
}
var notificationEvent2 = moira.NotificationEvent{
	Timestamp: time.Now().Unix(),
	State:     "OK",
	OldState:  "WARN",
	TriggerID: uuid.Must(uuid.NewV4()).String(),
	Metric:    "my.metric1",
}
