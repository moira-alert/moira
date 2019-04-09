package redis

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("ScheduledNotification manipulation", t, func(c C) {
		now := time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + 3600,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - 3600,
		}

		Convey("Test add and get by pages", t, func(c C) {
			addNotifications(c, dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, total, err := dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 3)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, 0)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 3)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err = dataBase.GetNotifications(1, 2)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 3)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})
		})

		Convey("Test fetch notifications", t, func(c C) {
			actual, err := dataBase.FetchNotifications(now - 3600)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 2)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, err = dataBase.FetchNotifications(now + 3600)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 0)
			c.So(actual, ShouldResemble, make([]*moira.ScheduledNotification, 0))
		})

		Convey("Test remove notifications by key", t, func(c C) {
			now := time.Now().Unix()
			id1 := "id1"
			notification1 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
			}
			notification2 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
			}
			notification3 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + 3600,
			}
			addNotifications(c, dataBase, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 3)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now), id1, id1}, ""))
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 2)

			actual, total, err = dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 1)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now+3600), id1, id1}, ""))
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 1)

			actual, total, err = dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 0)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test remove all notifications", t, func(c C) {
			now := time.Now().Unix()
			id1 := "id1"
			notification1 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
			}
			notification2 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
			}
			notification3 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + 3600,
			}
			addNotifications(c, dataBase, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 3)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			err = dataBase.RemoveAllNotifications()
			c.So(err, ShouldBeNil)

			actual, total, err = dataBase.GetNotifications(0, -1)
			c.So(err, ShouldBeNil)
			c.So(total, ShouldEqual, 0)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600)
			c.So(err, ShouldBeNil)
			c.So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})
	})
}

func addNotifications(c C, dataBase moira.Database, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		err := dataBase.AddNotification(&notification)
		c.So(err, ShouldBeNil)
	}
}

func TestScheduledNotificationErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Should throw error when no connection", t, func(c C) {
		actual1, total, err := dataBase.GetNotifications(0, 1)
		c.So(actual1, ShouldBeNil)
		c.So(total, ShouldEqual, 0)
		c.So(err, ShouldNotBeNil)

		total, err = dataBase.RemoveNotification("123")
		c.So(err, ShouldNotBeNil)
		c.So(total, ShouldEqual, 0)

		actual2, err := dataBase.FetchNotifications(0)
		c.So(err, ShouldNotBeNil)
		c.So(actual2, ShouldBeNil)

		notification := moira.ScheduledNotification{}
		err = dataBase.AddNotification(&notification)
		c.So(err, ShouldNotBeNil)

		err = dataBase.AddNotifications([]*moira.ScheduledNotification{&notification}, 0)
		c.So(err, ShouldNotBeNil)

		err = dataBase.RemoveAllNotifications()
		c.So(err, ShouldNotBeNil)
	})
}
