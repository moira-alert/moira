package redis

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/notifier"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("ScheduledNotification manipulation", t, func() {
		now := time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
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
			actual, err := dataBase.FetchNotifications(now-dataBase.GetDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, err = dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, make([]*moira.ScheduledNotification, 0))
		})

		Convey("Test fetch notifications limit 0", func() {
			actual, err := dataBase.FetchNotifications(now-dataBase.GetDelayedTimeInSeconds(), 0) //nolint
			So(err, ShouldBeError)
			So(actual, ShouldBeNil) //nolint
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
				Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
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

			total, err = dataBase.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now+dataBase.GetDelayedTimeInSeconds()), id1, id1}, "")) //nolint
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)

			actual, total, err = dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
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
				Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
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

			actual, err = dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})
	})
}

func addNotifications(dataBase *DbConnector, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		err := dataBase.AddNotification(&notification)
		So(err, ShouldBeNil)
	}
}

func TestScheduledNotificationErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()

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
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("FetchNotifications manipulation", t, func() {
		now := time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test fetch notifications with limit if all notifications has diff timestamp", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), 1) //nolint
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
			actual, err := dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), 4) //nolint
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
			actual, err := dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), 200000) //nolint
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
			actual, err := dataBase.FetchNotifications(now+dataBase.GetDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
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
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("notificationsCount in db", t, func() {
		now := time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test all notification with different ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.notificationsCount(now + dataBase.GetDelayedTimeInSeconds()*2)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(3))

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test get 0 notification with ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.notificationsCount(now - dataBase.GetDelayedTimeInSeconds()*2)
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
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
			"test2": {},
		},
		Timestamp: 110,
	}, moira.TriggerSourceNotSet)

	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
		},
		Timestamp: 110,
	}, moira.TriggerSourceNotSet)

	Convey("notificationsCount in db", t, func() {
		now := time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification4 := moira.ScheduledNotification{
			SendFail:  4,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		// create delayed notifications
		notificationNew2 := moira.ScheduledNotification{
			SendFail: 5,
			Trigger: moira.TriggerData{
				ID: "test1",
			},
			Event: moira.NotificationEvent{
				Metric: "test1",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 1,
			CreatedAt: now,
		}
		notificationNew3 := moira.ScheduledNotification{
			SendFail: 6,
			Trigger: moira.TriggerData{
				ID: "test1",
			},
			Event: moira.NotificationEvent{
				Metric: "test2",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 2,
			CreatedAt: now,
		}
		notificationNew4 := moira.ScheduledNotification{
			SendFail: 7,
			Trigger: moira.TriggerData{
				ID: "test2",
			},
			Event: moira.NotificationEvent{
				Metric: "test1",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 3,
			CreatedAt: now,
		}

		isLimit := true

		Convey("Test all notification with ts and limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), 1, isLimit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test 0 notification with ts and limit in empty db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), 10, isLimit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notification with ts and big limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), 100, isLimit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notification with ts and small limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), 3, isLimit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notification with ts and limit = count", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), 3, isLimit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with ts, big limit in db and deleted trigger", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
						"test2": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, 100, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with ts, limit = count in db and metric on meintenance", func() {
			_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test1": {},
					"test2": {
						Timestamp:   100,
						Maintenance: 130,
					},
				},
				Timestamp: 110,
			}, moira.TriggerSourceNotSet)
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
						"test2": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, 4, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew2})

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3, &notificationNew4})
			So(count, ShouldEqual, 2)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with ts, with small limit in db and trigger on meintenance", func() {
			_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test1": {},
					"test2": {},
				},
				Maintenance: 130,
				Timestamp:   110,
			}, moira.TriggerSourceNotSet)
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
						"test2": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, 3, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew})

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew2, &notificationNew3, &notificationNew4})
			So(count, ShouldEqual, 3)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test that returned valid notifications are sorted by timestamp with big ts", func() {
			notificationNew5 := moira.ScheduledNotification{
				SendFail: 8,
				Trigger: moira.TriggerData{
					ID: "test1",
				},
				Event: moira.NotificationEvent{
					Metric: "test1",
				},
				Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 4,
				CreatedAt: now + dataBase.GetDelayedTimeInSeconds(),
			}

			notificationNew6 := moira.ScheduledNotification{
				SendFail: 9,
				Trigger: moira.TriggerData{
					ID: "test1",
				},
				Event: moira.NotificationEvent{
					Metric: "test1",
				},
				Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 5,
				CreatedAt: now,
			}

			notificationNew7 := moira.ScheduledNotification{
				SendFail: 10,
				Trigger: moira.TriggerData{
					ID: "test1",
				},
				Event: moira.NotificationEvent{
					Metric: "test1",
				},
				Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 6,
				CreatedAt: now,
			}

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4, notificationNew5, notificationNew6, notificationNew7})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+6, 100, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew2, &notificationNew3, &notificationNew4, &notificationNew5, &notificationNew6})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestLimitNotifications(t *testing.T) { //nolint
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
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
			"test2": {},
		},
		Timestamp: 110,
	}, moira.TriggerSourceNotSet)

	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
		},
		Timestamp: 110,
	}, moira.TriggerSourceNotSet)

	Convey("fetchNotificationsNoLimit manipulation", t, func() {
		now := time.Now().Unix()

		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification4 := moira.ScheduledNotification{
			SendFail:  4,
			Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		// create delayed notifications
		notificationNew2 := moira.ScheduledNotification{
			SendFail: 5,
			Trigger: moira.TriggerData{
				ID: "test1",
			},
			Event: moira.NotificationEvent{
				Metric: "test1",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 1,
			CreatedAt: now,
		}
		notificationNew3 := moira.ScheduledNotification{
			SendFail: 6,
			Trigger: moira.TriggerData{
				ID: "test1",
			},
			Event: moira.NotificationEvent{
				Metric: "test2",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 2,
			CreatedAt: now,
		}
		notificationNew4 := moira.ScheduledNotification{
			SendFail: 7,
			Trigger: moira.TriggerData{
				ID: "test2",
			},
			Event: moira.NotificationEvent{
				Metric: "test1",
			},
			Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 3,
			CreatedAt: now,
		}

		var limit int64
		var isLimit bool

		Convey("Test all notifications with diff ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test zero notifications in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with various ts in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4, &notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with deleted trigger", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
						"test2": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew4})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with metric on maintenance", func() {
			_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test1": {},
					"test2": {
						Timestamp:   100,
						Maintenance: 130,
					},
				},
				Timestamp: 110,
			}, moira.TriggerSourceNotSet)
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
						"test2": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew2, &notificationNew4})

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3})
			So(count, ShouldEqual, 1)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with trigger on maintenance", func() {
			_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
				Metrics: map[string]moira.MetricState{
					"test1": {},
				},
				Maintenance: 130,
				Timestamp:   110,
			}, moira.TriggerSourceNotSet)
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
					},
					Timestamp: 110,
				}, moira.TriggerSourceNotSet)
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew2, &notificationNew3})

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew4})
			So(count, ShouldEqual, 1)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test that returned valid notifications are sorted by timestamp", func() {
			notificationNew5 := moira.ScheduledNotification{
				SendFail: 8,
				Trigger: moira.TriggerData{
					ID: "test1",
				},
				Event: moira.NotificationEvent{
					Metric: "test1",
				},
				Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 4,
				CreatedAt: now + dataBase.GetDelayedTimeInSeconds(),
			}

			notificationNew6 := moira.ScheduledNotification{
				SendFail: 9,
				Trigger: moira.TriggerData{
					ID: "test1",
				},
				Event: moira.NotificationEvent{
					Metric: "test1",
				},
				Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 5,
				CreatedAt: now,
			}

			addNotifications(dataBase, []moira.ScheduledNotification{notificationNew, notificationNew2, notificationNew3, notificationNew4, notificationNew5, notificationNew6})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+5, limit, isLimit)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew, &notificationNew2, &notificationNew3, &notificationNew4, &notificationNew5, &notificationNew6})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestFilterNotificationsByDelay(t *testing.T) {
	Convey("filterNotificationsByDelay manipulations", t, func() {
		notification1 := &moira.ScheduledNotification{
			Timestamp: 105,
			CreatedAt: 100,
		}
		notification2 := &moira.ScheduledNotification{
			Timestamp: 110,
			CreatedAt: 100,
		}
		notification3 := &moira.ScheduledNotification{
			Timestamp: 120,
			CreatedAt: 100,
		}

		Convey("Test with zero notifications", func() {
			notifications := []*moira.ScheduledNotification{}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 1)
			So(delayed, ShouldResemble, []*moira.ScheduledNotification{})
			So(notDelayed, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test with zero delayed notifications", func() {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 50)
			So(delayed, ShouldResemble, []*moira.ScheduledNotification{})
			So(notDelayed, ShouldResemble, notifications)
		})

		Convey("Test with zero not delayed notifications", func() {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 2)
			So(delayed, ShouldResemble, notifications)
			So(notDelayed, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test with one delayed and two not delayed notifications", func() {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 15)
			So(delayed, ShouldResemble, []*moira.ScheduledNotification{notification3})
			So(notDelayed, ShouldResemble, []*moira.ScheduledNotification{notification1, notification2})
		})
	})
}

func TestGetNotificationsTriggerChecks(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
		Timestamp: 1,
	}, moira.TriggerSourceNotSet)
	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Timestamp: 2,
	}, moira.TriggerSourceNotSet)

	Convey("getNotificationsTriggerChecks manipulations", t, func() {
		notification1 := &moira.ScheduledNotification{
			Trigger: moira.TriggerData{
				ID: "test1",
			},
		}
		notification2 := &moira.ScheduledNotification{
			Trigger: moira.TriggerData{
				ID: "test1",
			},
		}
		notification3 := &moira.ScheduledNotification{
			Trigger: moira.TriggerData{
				ID: "test2",
			},
		}

		Convey("Test with zero notifications", func() {
			notifications := []*moira.ScheduledNotification{}
			triggerChecks, err := dataBase.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test with correct notifications", func() {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := dataBase.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{
				{
					Timestamp:               1,
					MetricsToTargetRelation: map[string]string{},
				},
				{
					Timestamp:               1,
					MetricsToTargetRelation: map[string]string{},
				},
				{
					Timestamp:               2,
					MetricsToTargetRelation: map[string]string{},
				},
			})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with removed test1 trigger check", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Timestamp: 1,
				}, moira.TriggerSourceNotSet)
			}()

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := dataBase.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{nil, nil, {Timestamp: 2, MetricsToTargetRelation: map[string]string{}}})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with removed all trigger checks", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			dataBase.RemoveTriggerLastCheck("test2") //nolint

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := dataBase.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{nil, nil, nil})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}
