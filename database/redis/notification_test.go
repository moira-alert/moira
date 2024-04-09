package redis

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/notifier"
	"github.com/stretchr/testify/assert"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("ScheduledNotification manipulation", t, func() {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test add and get by pages", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err = database.GetNotifications(0, 0)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err = database.GetNotifications(1, 2)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})
		})

		Convey("Test fetch notifications", func() {
			actual, err := database.FetchNotifications(now-database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, err = database.FetchNotifications(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			actual, total, err = database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, make([]*moira.ScheduledNotification, 0))
		})

		Convey("Test fetch notifications limit 0", func() {
			actual, err := database.FetchNotifications(now-database.getDelayedTimeInSeconds(), 0) //nolint
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
				Timestamp: now + database.getDelayedTimeInSeconds(),
			}
			addNotifications(database, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			total, err = database.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now), id1, id1}, ""))
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)

			actual, total, err = database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification3})

			total, err = database.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now+database.getDelayedTimeInSeconds()), id1, id1}, "")) //nolint
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)

			actual, total, err = database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = database.FetchNotifications(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
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
				Timestamp: now + database.getDelayedTimeInSeconds(),
			}
			addNotifications(database, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 3)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification1, &notification2, &notification3})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)

			actual, total, err = database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})

			actual, err = database.FetchNotifications(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
		})
	})
}

func addNotifications(database *DbConnector, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		err := database.AddNotification(&notification)
		So(err, ShouldBeNil)
	}
}

func TestScheduledNotificationErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabaseWithIncorrectConfig(logger)
	database.Flush()
	defer database.Flush()

	Convey("Should throw error when no connection", t, func() {
		actual1, total, err := database.GetNotifications(0, 1)
		So(actual1, ShouldBeNil)
		So(total, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		total, err = database.RemoveNotification("123")
		So(err, ShouldNotBeNil)
		So(total, ShouldEqual, 0)

		actual2, err := database.FetchNotifications(0, notifier.NotificationsLimitUnlimited)
		So(err, ShouldNotBeNil)
		So(actual2, ShouldBeNil)

		notification := moira.ScheduledNotification{}
		err = database.AddNotification(&notification)
		So(err, ShouldNotBeNil)

		err = database.AddNotifications([]*moira.ScheduledNotification{&notification}, 0)
		So(err, ShouldNotBeNil)

		err = database.RemoveAllNotifications()
		So(err, ShouldNotBeNil)
	})
}

func TestFetchNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("FetchNotifications manipulation", t, func() {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test fetch notifications with limit if all notifications has diff timestamp", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(now+database.getDelayedTimeInSeconds(), 1) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 2)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications with limit little bit greater than count if all notifications has diff timestamp", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(now+database.getDelayedTimeInSeconds(), 4) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationNew})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications with limit greater than count if all notifications has diff timestamp", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(now+database.getDelayedTimeInSeconds(), 200000) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
			So(total, ShouldEqual, 0)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test fetch notifications without limit", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

			actual, total, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{})
			So(total, ShouldEqual, 0)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestGetNotificationsInTxWithLimit(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	client := *database.client
	ctx := database.context

	Convey("Test getNotificationsInTxWithLimit", t, func() {
		var limit int64 = 0
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test with zero notifications without limit", func() {
			addNotifications(database, []moira.ScheduledNotification{})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+database.getDelayedTimeInSeconds()*2, notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications without limit", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+database.getDelayedTimeInSeconds()*2, notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with limit != count", func() {
			limit = 1
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+database.getDelayedTimeInSeconds()*2, limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with limit = count", func() {
			limit = 3
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(ctx, tx, now+database.getDelayedTimeInSeconds()*2, limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestGetLimitedNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	client := *database.client
	ctx := database.context

	Convey("Test getLimitedNotifications", t, func() {
		var limit int64
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}

		Convey("Test all notifications with different timestamps without limit", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, notifier.NotificationsLimitUnlimited, notifications)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with different timestamps and limit", func() {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, limit, notifications)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with same timestamp and limit", func() {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now
			defer func() {
				notificationNew.Timestamp = now + database.getDelayedTimeInSeconds()
				notificationOld.Timestamp = now - database.getDelayedTimeInSeconds()
			}()

			addNotifications(database, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, limit, notifications)
				So(err, ShouldBeNil)
				assert.ElementsMatch(t, actual, expected)
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test not all notifications with same timestamp and limit", func() {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now
			defer func() {
				notificationNew.Timestamp = now + database.getDelayedTimeInSeconds()
				notificationOld.Timestamp = now - database.getDelayedTimeInSeconds()
			}()

			addNotifications(database, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(ctx, tx, limit, notifications)
				So(err, ShouldBeNil)
				assert.ElementsMatch(t, actual, expected)
				return nil
			}, notifierNotificationsKey)
			So(err, ShouldBeNil)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestFilterNotificationsByState(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()
	defaultSourceNotSetCluster := moira.MakeClusterKey(moira.TriggerSourceNotSet, moira.DefaultCluster)

	notificationOld := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		Event: moira.NotificationEvent{
			Metric: "test",
		},
		SendFail:  1,
		Timestamp: now - database.getDelayedTimeInSeconds(),
		CreatedAt: now - database.getDelayedTimeInSeconds(),
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
		Timestamp: now + database.getDelayedTimeInSeconds(),
		CreatedAt: now,
	}

	_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)

	_ = database.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test": {},
		},
	}, defaultSourceNotSetCluster)

	Convey("Test filter notifications by state", t, func() {
		Convey("With empty notifications", func() {
			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveOld, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveNew, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With all valid notifications", func() {
			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveOld, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveNew, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With removed check data", func() {
			database.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)
			}()

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew})
			So(types.ToResaveOld, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveNew, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With metric on maintenance", func() {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notification, notificationNew})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(len(types.ToResaveOld), ShouldResemble, 1)
			So(types.ToResaveOld[0], ShouldResemble, notificationOld)
			So(len(types.ToResaveNew), ShouldResemble, 1)
			So(types.ToResaveNew[0].SendFail, ShouldResemble, notificationOld.SendFail)
		})

		Convey("With trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{})
			So(len(types.ToResaveOld), ShouldResemble, 1)
			So(types.ToResaveOld[0], ShouldResemble, notificationNew)
			So(len(types.ToResaveNew), ShouldResemble, 1)
			So(types.ToResaveNew[0].SendFail, ShouldResemble, notificationNew.SendFail)
		})
	})
}

func TestHandleNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()
	defaultSourceNotSetCluster := moira.MakeClusterKey(moira.TriggerSourceNotSet, moira.DefaultCluster)

	notificationOld := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test2",
		},
		SendFail:  1,
		Timestamp: now - database.getDelayedTimeInSeconds(),
		CreatedAt: now - database.getDelayedTimeInSeconds(),
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
		Timestamp: now + database.getDelayedTimeInSeconds(),
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
		Timestamp: now - database.getDelayedTimeInSeconds() + 1,
		CreatedAt: now - 2*database.getDelayedTimeInSeconds(),
	}
	notificationNew2 := &moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		Event: moira.NotificationEvent{
			Metric: "test",
		},
		SendFail:  5,
		Timestamp: now + database.getDelayedTimeInSeconds() + 1,
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
		Timestamp: now + database.getDelayedTimeInSeconds() + 2,
		CreatedAt: now,
	}

	_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)

	_ = database.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test": {},
		},
	}, defaultSourceNotSetCluster)

	Convey("Test handle notifications", t, func() {
		Convey("Without delayed notifications", func() {
			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			var toResaveNotificationsExpected []*moira.ScheduledNotification
			So(types.ToResaveOld, ShouldResemble, toResaveNotificationsExpected)
			So(types.ToResaveNew, ShouldResemble, toResaveNotificationsExpected)
		})

		Convey("With both delayed and not delayed valid notifications", func() {
			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.ToResaveOld, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveNew, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With both delayed and not delayed notifications and removed check data", func() {
			database.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)
			}()

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationNew2, notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.ToResaveOld, ShouldResemble, []*moira.ScheduledNotification{})
			So(types.ToResaveNew, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("With both delayed and not delayed valid notifications and metric on maintenance", func() {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew, notificationNew2, notificationNew3})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notification, notificationNew, notificationNew2, notificationNew3})
			So(len(types.ToResaveNew), ShouldResemble, 1)
			So(types.ToResaveNew[0].SendFail, ShouldResemble, notificationOld2.SendFail)
			So(len(types.ToResaveOld), ShouldResemble, 1)
			So(types.ToResaveOld[0], ShouldResemble, notificationOld2)
		})

		Convey("With both delayed and not delayed valid notifications and trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(types.Valid, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(types.ToRemove, ShouldResemble, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3})
			So(len(types.ToResaveNew), ShouldResemble, 1)
			So(types.ToResaveNew[0].SendFail, ShouldResemble, notificationNew2.SendFail)
			So(len(types.ToResaveOld), ShouldResemble, 1)
			So(types.ToResaveOld[0], ShouldResemble, notificationNew2)
		})
	})
}

func TestNotificationsCount(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	Convey("notificationsCount in db", t, func() {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now - database.getDelayedTimeInSeconds(),
		}

		Convey("Test all notification with different ts in db", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(now + database.getDelayedTimeInSeconds()*2)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(3))

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test get 0 notification with ts in db", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(now - database.getDelayedTimeInSeconds()*2)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(0))

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test part notification in db with ts", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(now)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(2))

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test 0 notification in db", func() {
			addNotifications(database, []moira.ScheduledNotification{})
			actual, err := database.notificationsCount(now)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, int64(0))

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestFetchNotificationsDo(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	var limit int64

	defaultSourceNotSetCluster := moira.MakeClusterKey(moira.TriggerSourceNotSet, moira.DefaultCluster)

	_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
		},
	}, defaultSourceNotSetCluster)

	_ = database.SetTriggerLastCheck("test2", &moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"test1": {},
			"test2": {},
		},
	}, defaultSourceNotSetCluster)

	now := time.Now().Unix()
	notificationOld := moira.ScheduledNotification{
		SendFail:  1,
		Timestamp: now - database.getDelayedTimeInSeconds() + 1,
		CreatedAt: now - database.getDelayedTimeInSeconds() + 1,
	}
	notification4 := moira.ScheduledNotification{
		SendFail:  2,
		Timestamp: now - database.getDelayedTimeInSeconds() + 2,
		CreatedAt: now - database.getDelayedTimeInSeconds() + 2,
	}
	notificationNew := moira.ScheduledNotification{
		SendFail:  3,
		Timestamp: now + database.getDelayedTimeInSeconds(),
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
		Timestamp: now - database.getDelayedTimeInSeconds() + 3,
		CreatedAt: now - 2*database.getDelayedTimeInSeconds(),
	}
	notificationNew2 := moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID: "test1",
		},
		Event: moira.NotificationEvent{
			Metric: "test1",
		},
		SendFail:  6,
		Timestamp: now + database.getDelayedTimeInSeconds() + 1,
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
		Timestamp: now + database.getDelayedTimeInSeconds() + 2,
		CreatedAt: now,
	}

	Convey("Test fetchNotificationsDo", t, func() {
		Convey("Test all notifications with diff ts in db", func() {
			Convey("With limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
				limit = 1
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld})

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew})

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test zero notifications with ts in empty db", func() {
			Convey("With limit", func() {
				addNotifications(database, []moira.ScheduledNotification{})
				limit = 10
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(database, []moira.ScheduledNotification{})
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{})

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test all notification with ts and without limit in db", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), notifier.NotificationsLimitUnlimited)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4, &notification, &notificationNew})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test all notifications with ts and big limit in db", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			limit = 100
			actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with ts and small limit in db", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			limit = 3
			actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification4})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with ts and limit = count", func() {
			addNotifications(database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			limit = 3
			actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds(), limit) //nolint
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notification})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test delayed notifications with ts and deleted trigger", func() {
			database.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
					},
				}, defaultSourceNotSetCluster)
			}()

			Convey("With big limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 100
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3})
				So(count, ShouldEqual, 1)

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
				So(count, ShouldEqual, 0)

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test notifications with ts and metric on maintenance", func() {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": 0}, nil, "test", 100)                          //nolint

			Convey("With limit = count", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 6
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notificationNew3})
				So(count, ShouldEqual, 1)

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("Without limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
				So(allNotifications[0].SendFail, ShouldResemble, notificationNew3.SendFail)

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})
		})

		Convey("Test delayed notifications with ts and trigger on maintenance", func() {
			var triggerMaintenance int64 = time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			Convey("With small limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				limit = 3
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, limit)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{&notification, &notificationNew, &notificationNew2, &notificationNew3})
				So(count, ShouldEqual, 4)

				err = database.RemoveAllNotifications()
				So(err, ShouldBeNil)
			})

			Convey("without limit", func() {
				addNotifications(database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsDo(now+database.getDelayedTimeInSeconds()+3, notifier.NotificationsLimitUnlimited)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3})

				allNotifications, count, err := database.GetNotifications(0, -1)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
				So(allNotifications[0].SendFail, ShouldResemble, notificationNew2.SendFail)

				err = database.RemoveAllNotifications()
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
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()
	defaultSourceNotSetCluster := moira.MakeClusterKey(moira.TriggerSourceNotSet, moira.DefaultCluster)

	_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
		Timestamp: 1,
	}, defaultSourceNotSetCluster)
	_ = database.SetTriggerLastCheck("test2", &moira.CheckData{
		Timestamp: 2,
	}, defaultSourceNotSetCluster)

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
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test with correct notifications", func() {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{
				{
					Timestamp:               1,
					MetricsToTargetRelation: map[string]string{},
					Clock:                   clock.NewSystemClock(),
				},
				{
					Timestamp:               1,
					MetricsToTargetRelation: map[string]string{},
					Clock:                   clock.NewSystemClock(),
				},
				{
					Timestamp:               2,
					MetricsToTargetRelation: map[string]string{},
					Clock:                   clock.NewSystemClock(),
				},
			})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with removed test1 trigger check", func() {
			database.RemoveTriggerLastCheck("test1") //nolint
			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
					Timestamp: 1,
				}, defaultSourceNotSetCluster)
			}()

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{nil, nil, {Timestamp: 2, MetricsToTargetRelation: map[string]string{}, Clock: clock.NewSystemClock()}})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test notifications with removed all trigger checks", func() {
			database.RemoveTriggerLastCheck("test1") //nolint
			database.RemoveTriggerLastCheck("test2") //nolint

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			So(err, ShouldBeNil)
			So(triggerChecks, ShouldResemble, []*moira.CheckData{nil, nil, nil})

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestResaveNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	client := database.client
	ctx := database.context
	pipe := (*client).TxPipeline()

	Convey("Test resaveNotifications", t, func() {
		notificationOld1 := &moira.ScheduledNotification{
			Timestamp: 1,
		}
		notificationOld2 := &moira.ScheduledNotification{
			Timestamp: 2,
		}
		notificationOld3 := &moira.ScheduledNotification{
			Timestamp: 3,
		}

		notificationNew1 := &moira.ScheduledNotification{
			Timestamp: 4,
		}
		notificationNew2 := &moira.ScheduledNotification{
			Timestamp: 5,
		}
		notificationNew3 := &moira.ScheduledNotification{
			Timestamp: 6,
		}

		Convey("Test resave with zero notifications", func() {
			affected, err := database.resaveNotifications(ctx, pipe, []*moira.ScheduledNotification{}, []*moira.ScheduledNotification{})
			So(err, ShouldBeNil)
			So(affected, ShouldResemble, 0)

			allNotifications, count, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
			So(count, ShouldEqual, 0)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test resave notifications with empty database", func() {
			affected, err := database.resaveNotifications(ctx, pipe, []*moira.ScheduledNotification{notificationOld1, notificationOld2}, []*moira.ScheduledNotification{notificationNew1, notificationNew2})
			So(err, ShouldBeNil)
			So(affected, ShouldResemble, 2)

			allNotifications, count, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notificationNew1, notificationNew2})
			So(count, ShouldEqual, 2)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test resave one notification when other notifications exist in the database", func() {
			addNotifications(database, []moira.ScheduledNotification{*notificationOld1, *notificationOld3})

			affected, err := database.resaveNotifications(ctx, pipe, []*moira.ScheduledNotification{notificationOld2}, []*moira.ScheduledNotification{notificationNew2})
			So(err, ShouldBeNil)
			So(affected, ShouldResemble, 1)

			allNotifications, count, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notificationOld1, notificationOld3, notificationNew2})
			So(count, ShouldEqual, 3)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})

		Convey("Test resave all notifications", func() {
			addNotifications(database, []moira.ScheduledNotification{*notificationOld1, *notificationOld2, *notificationOld3})

			affected, err := database.resaveNotifications(ctx, pipe, []*moira.ScheduledNotification{notificationOld1, notificationOld2, notificationOld3}, []*moira.ScheduledNotification{notificationNew1, notificationNew2, notificationNew3})
			So(err, ShouldBeNil)
			So(affected, ShouldResemble, 6)

			allNotifications, count, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notificationNew1, notificationNew2, notificationNew3})
			So(count, ShouldEqual, 3)

			err = database.RemoveAllNotifications()
			So(err, ShouldBeNil)
		})
	})
}

func TestRemoveNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()
	defer database.Flush()

	client := database.client
	ctx := database.context
	pipe := (*client).TxPipeline()

	notification1 := &moira.ScheduledNotification{
		Timestamp: 1,
	}
	notification2 := &moira.ScheduledNotification{
		Timestamp: 2,
	}
	notification3 := &moira.ScheduledNotification{
		Timestamp: 3,
	}

	Convey("Test removeNotifications", t, func() {
		Convey("Test remove empty notifications", func() {
			count, err := database.removeNotifications(ctx, pipe, []*moira.ScheduledNotification{})
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(countAllNotifications, ShouldEqual, 0)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test remove one notification", func() {
			addNotifications(database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(ctx, pipe, []*moira.ScheduledNotification{notification2})
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 1)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(countAllNotifications, ShouldEqual, 2)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notification1, notification3})
		})

		Convey("Test remove all notifications", func() {
			addNotifications(database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(ctx, pipe, []*moira.ScheduledNotification{notification1, notification2, notification3})
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 3)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(countAllNotifications, ShouldEqual, 0)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{})
		})

		Convey("Test remove a nonexistent notification", func() {
			notification4 := &moira.ScheduledNotification{
				Timestamp: 4,
			}
			addNotifications(database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(ctx, pipe, []*moira.ScheduledNotification{notification4})
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			So(err, ShouldBeNil)
			So(countAllNotifications, ShouldEqual, 3)
			So(allNotifications, ShouldResemble, []*moira.ScheduledNotification{notification1, notification2, notification3})
		})
	})
}
