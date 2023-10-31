package redis

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/notifier"
	"github.com/stretchr/testify/assert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("ScheduledNotification manipulation", t, func() {
		now = time.Now().Unix()
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
		now = time.Now().Unix()
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

func TestGetNotificationsInTxWithLimit(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	client := *dataBase.client
	ctx := dataBase.context

	Convey("Test getNotificationsInTxWithLimit", t, func() {
		var limit int64 = 0
		now = time.Now().Unix()
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

		Convey("Test with zero notifications without limit", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+dataBase.GetDelayedTimeInSeconds()*2, nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications without limit", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+dataBase.GetDelayedTimeInSeconds()*2, nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with limit != count", func() {
			limit = 1
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+dataBase.GetDelayedTimeInSeconds()*2, &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with limit = count", func() {
			limit = 3
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+dataBase.GetDelayedTimeInSeconds()*2, &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestGetLimitedNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	client := *dataBase.client
	ctx := dataBase.context

	Convey("Test getLimitedNotifications", t, func() {
		var limit int64 = 0
		now = time.Now().Unix()
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

		Convey("Test all notifications with different timestamps without limit", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, nil, notifications)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with different timestamps and limit", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, &limit, notifications)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with same timestamp and limit", func() {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now
			defer func() {
				notificationNew.Timestamp = now + dataBase.GetDelayedTimeInSeconds()
				notificationOld.Timestamp = now - dataBase.GetDelayedTimeInSeconds()
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, &limit, notifications)
				So(err, ShouldBeNil)
				assert.ElementsMatch(t, actual, expected)
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test not all notifications with same timestamp and limit", func() {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now
			defer func() {
				notificationNew.Timestamp = now + dataBase.GetDelayedTimeInSeconds()
				notificationOld.Timestamp = now - dataBase.GetDelayedTimeInSeconds()
			}()

			addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, &limit, notifications)
				So(err, ShouldBeNil)
				assert.ElementsMatch(t, actual, expected)
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestFilterNotificationsByState(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	notificationOld := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test",
		},
		SendFail:  1,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now - dataBase.GetDelayedTimeInSeconds(),
	}
	notification := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test1",
		},
		SendFail:  2,
		Timestamp: now,
		CreatedAt: now,
	}
	notificationNew := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		Event: moira.NotificationEvent{
			Metric: "test1",
		},
		SendFail:  3,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now,
	}

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{}, moira.TriggerSourceNotSet)

	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test": {},
		},
	}, moira.TriggerSourceNotSet)

	Convey("Test filter notifications by state", t, func() {
		Convey("With empty notifications", func() {
			types, err := dataBase.filterNotificationsByState([]*moira.ScheduledNotification{})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With all valid notifications", func() {
			types, err := dataBase.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With removed check data", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{}, moira.TriggerSourceNotSet)
			}()

			types, err := dataBase.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With metric on maintenance", func() {
			dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			updatedNotificationOld := *notificationOld
			updatedNotificationOld.Timestamp += dataBase.GetResaveTimeInSeconds()

			types, err := dataBase.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notification, notificationNew})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationOld})
		})

		Convey("With trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			updatedNotificationNew := *notificationNew
			updatedNotificationNew.Timestamp += dataBase.GetResaveTimeInSeconds()

			types, err := dataBase.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationNew})
		})
	})
}

func TestHandleNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	notificationOld := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		SendFail:  1,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now - dataBase.GetDelayedTimeInSeconds(),
	}
	notification := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		SendFail:  2,
		Timestamp: now,
		CreatedAt: now,
	}
	notificationNew := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		SendFail:  3,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now,
	}

	// create delayed notifications
	notificationOld2 := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test",
		},
		SendFail:  4,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds() + 1,
		CreatedAt: now - 2*dataBase.GetDelayedTimeInSeconds(),
	}
	notificationNew2 := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		Event: moira.NotificationEvent{
			Metric: "test",
		},
		SendFail:  5,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 1,
		CreatedAt: now,
	}
	notificationNew3 := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test2",
		},
		SendFail:  6,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 2,
		CreatedAt: now,
	}

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{}, moira.TriggerSourceNotSet)

	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test": {},
		},
	}, moira.TriggerSourceNotSet)

	Convey("Test handle notifications", t, func() {
		Convey("Without delayed notifications", func() {
			types, err := dataBase.handleNotifications([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			var toResaveExpected []*moira.ScheduledNotification
			So(types.toResave, ShouldResemble, toResaveExpected)
		})

		Convey("With both delayed and not delayed valid notifications", func() {
			types, err := dataBase.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With both delayed and not delayed notifications and removed check data", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{}, moira.TriggerSourceNotSet)
			}()

			types, err := dataBase.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew2, notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With both delayed and not delayed valid notifications and metric on maintenance", func() {
			dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			updatedNotificationOld2 := *notificationOld2
			updatedNotificationOld2.Timestamp += dataBase.GetResaveTimeInSeconds()

			types, err := dataBase.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld2, notificationOld, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationOld2})
		})

		Convey("With both delayed and not delayed valid notifications and trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			updatedNotificationNew2 := *notificationNew2
			updatedNotificationNew2.Timestamp += dataBase.GetResaveTimeInSeconds()

			types, err := dataBase.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.toRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew2, notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.toResave, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationNew2})
		})
	})
}

func TestNotificationsCount(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("notificationsCount in db", t, func() {
		now = time.Now().Unix()
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
			CreatedAt: now - dataBase.GetDelayedTimeInSeconds(),
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

func TestFetchNotificationsDo(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	var limit int64

	_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
		},
	}, moira.TriggerSourceNotSet)

	_ = dataBase.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
			"test2": {},
		},
	}, moira.TriggerSourceNotSet)

	now := time.Now().Unix()
	notificationOld := moira.ScheduledNotification{
		SendFail:  1,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds() + 1,
		CreatedAt: now - dataBase.GetDelayedTimeInSeconds() + 1,
	}
	notification4 := moira.ScheduledNotification{
		SendFail:  2,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds() + 2,
		CreatedAt: now - dataBase.GetDelayedTimeInSeconds() + 2,
	}
	notificationNew := moira.ScheduledNotification{
		SendFail:  3,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds(),
		CreatedAt: now,
	}
	notification := moira.ScheduledNotification{
		SendFail:  4,
		Timestamp: now,
		CreatedAt: now,
	}

	// create delayed notifications
	notificationOld2 := moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test1",
		},
		SendFail:  5,
		Timestamp: now - dataBase.GetDelayedTimeInSeconds() + 3,
		CreatedAt: now - 2*dataBase.GetDelayedTimeInSeconds(),
	}
	notificationNew2 := moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		Event: moira.NotificationEvent{
			Metric: "test1",
		},
		SendFail:  6,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 1,
		CreatedAt: now,
	}
	notificationNew3 := moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test2",
		},
		SendFail:  7,
		Timestamp: now + dataBase.GetDelayedTimeInSeconds() + 2,
		CreatedAt: now,
	}

	Convey("Test fetchNotificationsDo", t, func() {
		Convey("Test all notifications with diff ts in db", func() {
			Convey("With limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
				limit = 1
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test zero notifications with ts in empty db", func() {
			Convey("With limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{})
				limit = 10
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{})
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test all notification with ts and without limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), nil)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4, &notification, &notificationNew})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with ts and big limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			limit = 100
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), &limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with ts and small limit in db", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			limit = 3
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), &limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with ts and limit = count", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			limit = 3
			actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds(), &limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with ts and deleted trigger", func() {
			dataBase.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = dataBase.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
					},
				}, moira.TriggerSourceNotSet)
			}()

			Convey("With big limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 100
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3})
				So(count, ShouldEqual, 1)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
				So(count, ShouldEqual, 0)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test notifications with ts and metric on maintenance", func() {
			dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer dataBase.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": 0}, nil, "test", 100)                          //nolint

			updatedNotificationNew3 := notificationNew3
			updatedNotificationNew3.Timestamp += dataBase.GetResaveTimeInSeconds()

			Convey("With limit = count", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 6
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3})
				So(count, ShouldEqual, 1)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationNew3})
				So(count, ShouldEqual, 1)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test delayed notifications with ts and trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				dataBase.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			updatedNotificationNew2 := notificationNew2
			updatedNotificationNew2.Timestamp += dataBase.GetResaveTimeInSeconds()

			Convey("With small limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 3
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, &limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew, &notificationNew2, &notificationNew3})
				So(count, ShouldEqual, 4)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("without limit", func() {
				addNotifications(dataBase, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := dataBase.fetchNotificationsDo(now+dataBase.GetDelayedTimeInSeconds()+3, nil)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3})

				allNotifications, count, err := dataBase.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&updatedNotificationNew2})
				So(count, ShouldEqual, 1)

				err = dataBase.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
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

func TestSaveNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	client := dataBase.client
	ctx := dataBase.context
	pipe := (*client).TxPipeline()

	Convey("Test saveNotifications", t, func() {
		notification1 := &moira.ScheduledNotification{
			Timestamp: 1,
			SendFail:  1,
		}
		notification2 := &moira.ScheduledNotification{
			Timestamp: 2,
			SendFail:  2,
		}
		notification3 := &moira.ScheduledNotification{
			Timestamp: 3,
		}

		Convey("Test with zero notifications", func() {
			err := dataBase.saveNotifications(ctx, pipe, []*moira.ScheduledNotification{})
			So(err, ShouldBeNil)

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
			So(count, ShouldEqual, 0)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test save notifications with empty database", func() {
			err := dataBase.saveNotifications(ctx, pipe, []*moira.ScheduledNotification{notification1, notification2, notification3})
			So(err, ShouldBeNil)

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notification1, notification2, notification3})
			So(count, ShouldEqual, 3)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test save one notification when other notifications exist in the database", func() {
			addNotifications(dataBase, []moira.ScheduledNotification{*notification1, *notification3})

			err := dataBase.saveNotifications(ctx, pipe, []*moira.ScheduledNotification{notification2})
			So(err, ShouldBeNil)

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notification1, notification2, notification3})
			So(count, ShouldEqual, 3)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test save one notification when there is already one with the same timestamp", func() {
			newNotification := &moira.ScheduledNotification{
				Timestamp: 2,
				SendFail:  3,
			}
			addNotifications(dataBase, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			err := dataBase.saveNotifications(ctx, pipe, []*moira.ScheduledNotification{newNotification})
			So(err, ShouldBeNil)

			allNotifications, count, err := dataBase.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			assert.ElementsMatch(t, allNotifications, []*moira.ScheduledNotification{notification1, notification2, newNotification, notification3})
			So(count, ShouldEqual, 4)

			err = dataBase.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}
