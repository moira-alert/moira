package redis

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

var inputScheduledNotification = moira.ScheduledNotification{
	Event: moira.NotificationEvent{
		IsTriggerEvent: true,
		Timestamp:      time.Now().Unix(),
		Metric:         "some_metric",
		State:          moira.StateERROR,
		OldState:       moira.StateOK,
		TriggerID:      "1111-2222-33-4444-5555",
	},
	Trigger: moira.TriggerData{
		ID:   "1111-2222-33-4444-5555",
		Name: "Awesome Trigger",
		Desc: "No desc",
		Targets: []string{
			"some.metric.path",
		},
		WarnValue:  0.9,
		ErrorValue: 1.0,
		IsRemote:   false,
		Tags: []string{
			"TEST_TAG1",
			"TEST_TAG2",
		},
	},
	Contact: moira.ContactData{
		Type:  "slack",
		Value: "#auf_channel",
		ID:    "contact_id",
		User:  "user",
	},
	Plotting: moira.PlottingData{
		Enabled: false,
	},
	Throttled: false,
	SendFail:  1,
	Timestamp: time.Now().Unix(),
}

var eventsShouldBeInDb = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: inputScheduledNotification.Timestamp,
		Metric:    inputScheduledNotification.Event.Metric,
		State:     inputScheduledNotification.Event.State,
		OldState:  inputScheduledNotification.Event.OldState,
		TriggerID: inputScheduledNotification.Trigger.ID,
		ContactID: inputScheduledNotification.Contact.ID,
	},
}

func TestGetNotificationsByContactIdWithLimit(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)

	Convey("Notification history items manipulation", t, func() {
		dataBase.Flush()
		defer dataBase.Flush()

		Convey("While no data then notification items should be empty", func() {
			items, err := dataBase.GetNotificationsByContactIdWithLimit(
				"id",
				eventsShouldBeInDb[0].TimeStamp,
				eventsShouldBeInDb[0].TimeStamp)

			So(err, ShouldBeNil)
			So(items, ShouldHaveLength, 0)
		})

		Convey("Write event and check for success write", func() {
			err := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
			So(err, ShouldBeNil)

			Convey("Ensure that we can find event on +- 5 seconds interval", func() {
				eventFromDb, err := dataBase.GetNotificationsByContactIdWithLimit(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp+5)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event exactly by its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsByContactIdWithLimit(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event if 'from' border equals its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsByContactIdWithLimit(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp+5)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event if 'to' border equals its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsByContactIdWithLimit(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can't find event time borders don't fit event timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsByContactIdWithLimit(
					eventsShouldBeInDb[0].ContactID,
					928930626,
					992089026)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldNotResemble, eventsShouldBeInDb)
			})
		})
	})
}

func TestPushNotificationToHistory(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.notificationHistory.NotificationHistoryQueryLimit = 500

	Convey("Ensure that event would not have duplicates", t, func() {
		dataBase.Flush()
		defer dataBase.Flush()

		err1 := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
		So(err1, ShouldBeNil)

		err2 := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
		So(err2, ShouldBeNil)

		dbContent, err3 := dataBase.GetNotificationsByContactIdWithLimit(
			inputScheduledNotification.Contact.ID,
			inputScheduledNotification.Timestamp,
			inputScheduledNotification.Timestamp)

		So(err3, ShouldBeNil)
		So(dbContent, ShouldHaveLength, 1)
	})
}
