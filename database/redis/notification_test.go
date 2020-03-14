package redis

import (
	"fmt"
	"github.com/moira-alert/moira/notifier"
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

	Convey("ScheduledNotification manipulation", t, func() {
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

		Convey("Test add and get by pages", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, 0)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err = dataBase.GetNotifications(1, 2)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})
		})

		Convey("Test fetch notifications", func() {
			actual, err := dataBase.FetchNotifications(now - 3600, notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, err = dataBase.FetchNotifications(now + 3600, notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, make([]*moira.ScheduledNotification, 0))
		})

		Convey("Test fetch notifications limit 0", func() {
			actual, err := dataBase.FetchNotifications(now - 3600, 0)
			So(err, ShouldBeError)
			So(actual, ShouldBeNil )
		})

		Convey("Test remove notifications by key", func() {
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
			addNotifications(dataBase, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now), id1, id1}, ""))
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification3})

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now + 3600), id1, id1}, ""))
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600, notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test remove all notifications", func() {
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
			addNotifications(dataBase, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now + 3600, notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})
	})
}

func addNotifications(dataBase moira.Database, notifications []moira.ScheduledNotification) {
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

		actual2, err := dataBase.FetchNotifications(0, notifier.NotificationsLimitUnlimited)
		So(err, ShouldNotBeNil)
		So(actual2, ShouldBeNil)

		notification := moira.ScheduledNotification{}
		err = dataBase.AddNotification(&notification)
		So(err, ShouldNotBeNil)

		err = dataBase.AddNotifications([]*moira.ScheduledNotification{&notification}, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveAllNotifications()
		So(err, ShouldNotBeNil)
	})
}

func TestFetchNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("FetchNotifications manipulation", t, func() {
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

		Convey("Test fetch notifications with limit if all notifications has diff timestamp", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.FetchNotifications(now + 6000, 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications with limit little bit greater than count if all notifications has diff timestamp", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.FetchNotifications(now + 6000, 4)
			So(err, ShouldBeNil)

			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications with limit greater than count if all notifications has diff timestamp", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.FetchNotifications(now + 6000, 200000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
			So(total, ShouldEqual, 0)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications without limit", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.FetchNotifications(now + 6000, notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
			So(total, ShouldEqual, 0)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestNotificationsCount(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("notificationsCount in db", t, func() {
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

		Convey("Test all notification with different ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.notificationsCount(now + 6000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(3))

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test get 0 notification with ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.notificationsCount(now - 7000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(0))

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test part notification in db with ts", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.notificationsCount(now)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(2))

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test 0 notification in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			actual, err := dataBase.notificationsCount(now)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(0))

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestFetchNotificationsWithLimitDo(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("notificationsCount in db", t, func() {
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
		notification4 := moira.ScheduledNotification{
			SendFail:  4,
			Timestamp: now - 3600,
		}

		Convey("Test all notification with ts and limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsWithLimitDo(now + 6000, 1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test 0 notification with ts and limit in empty db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			actual, err := dataBase.fetchNotificationsWithLimitDo(now + 6000, 10)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notification with ts and big limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsWithLimitDo(now + 6000, 100)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notification with ts and small limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := dataBase.fetchNotificationsWithLimitDo(now + 6000, 3)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notification with ts and limit = count", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsWithLimitDo(now + 6000, 3)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestLimitNotifications(t *testing.T)  {
	Convey("limitNotifications manipulation", t, func() {
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
		notification4 := moira.ScheduledNotification{
			SendFail:  4,
			Timestamp: now - 3600,
		}
		notification5 := moira.ScheduledNotification{
			SendFail:  5,
			Timestamp: now + 3600,
		}

		Convey("Test limit Notifications zero notifications", func() {
			notifications := []*moira.ScheduledNotification{}
			actual := limitNotifications(notifications)
			So(actual, ShouldResemble, notifications)
		})

		Convey("Test limit Notifications notifications diff ts", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			actual := limitNotifications(notifications)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})
		})

		Convey("Test limit Notifications all notifications same ts", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification4}
			actual := limitNotifications(notifications)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})
		})

		Convey("Test limit Notifications 1 notifications diff ts and 2 same ts", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notificationNew, &notification5}
			actual := limitNotifications(notifications)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})
		})

		Convey("Test limit Notifications 2 notifications same ts and 1 diff ts", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification4, &notification5}
			actual := limitNotifications(notifications)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})
		})
	})
}

func TestFetchNotificationsNoLimit(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("fetchNotificationsNoLimit manipulation", t, func() {
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
		notification4 := moira.ScheduledNotification{
			SendFail:  4,
			Timestamp: now - 3600,
		}

		Convey("Test all notifications with diff ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsNoLimit(now + 6000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test zero notifications in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			actual, err := dataBase.fetchNotificationsNoLimit(now + 6000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with various ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := dataBase.fetchNotificationsNoLimit(now + 6000)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4, &notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}
