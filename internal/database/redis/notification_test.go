package redis

import (
	"fmt"
	"strings"
	"testing"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("ScheduledNotification manipulation", t, func() {
		now := time.Now().Unix()
		notificationNew := moira2.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + 3600,
		}
		notification := moira2.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
		}
		notificationOld := moira2.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - 3600,
		}

		Convey("Test add and get by pages", func() {
			addNotifications(dataBase, []moira2.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, 0)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notificationOld})

			actual, total, err = dataBase.GetNotifications(1, 2)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification, &notificationNew})
		})

		Convey("Test fetch notifications", func() {
			actual, err := dataBase.FetchNotifications(now - 3600)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notificationOld})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification, &notificationNew})

			actual, err = dataBase.FetchNotifications(now + 3600)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, make([]*moira2.ScheduledNotification, 0))
		})

		Convey("Test remove notifications by key", func() {
			now := time.Now().Unix()
			id1 := "id1"
			notification1 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
			}
			notification2 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
			}
			notification3 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + 3600,
			}
			addNotifications(dataBase, []moira2.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification1, &notification2, &notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now), id1, id1}, ""))
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now+3600), id1, id1}, ""))
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{})
		})

		Convey("Test remove all notifications", func() {
			now := time.Now().Unix()
			id1 := "id1"
			notification1 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
			}
			notification2 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
			}
			notification3 := moira2.ScheduledNotification{
				Contact:   moira2.ContactData{ID: id1},
				Event:     moira2.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + 3600,
			}
			addNotifications(dataBase, []moira2.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{&notification1, &notification2, &notification3})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira2.ScheduledNotification{})
		})
	})
}

func addNotifications(dataBase moira2.Database, notifications []moira2.ScheduledNotification) {
	for _, notification := range notifications {
		err := dataBase.AddNotification(&notification)
		So(err, ShouldBeNil)
	}
}

func TestScheduledNotificationErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Should throw error when no connection", t, func() {
		actual1, total, err := dataBase.GetNotifications(0, 1)
		So(actual1, ShouldBeNil)
		So(total, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		total, err = dataBase.RemoveNotification("123")
		So(err, ShouldNotBeNil)
		So(total, ShouldEqual, 0)

		actual2, err := dataBase.FetchNotifications(0)
		So(err, ShouldNotBeNil)
		So(actual2, ShouldBeNil)

		notification := moira2.ScheduledNotification{}
		err = dataBase.AddNotification(&notification)
		So(err, ShouldNotBeNil)

		err = dataBase.AddNotifications([]*moira2.ScheduledNotification{&notification}, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveAllNotifications()
		So(err, ShouldNotBeNil)
	})
}
