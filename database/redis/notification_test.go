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
	"github.com/stretchr/testify/require"
)

func TestScheduledNotification(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	t.Run("ScheduledNotification manipulation", func(t *testing.T) {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test add and get by pages", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

			actual, total, err = database.GetNotifications(0, 0)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)

			actual, total, err = database.GetNotifications(1, 2)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification, &notificationNew}, actual)
		})

		t.Run("Test fetch notifications", func(t *testing.T) {
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now-database.getDelayedTimeInSeconds(), notificationsLimitUnlimited) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)

			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 2, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification, &notificationNew}, actual)

			actual, err = database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notification, &notificationNew}, actual)

			actual, total, err = database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 0, total)
			require.Equal(t, actual, make([]*moira.ScheduledNotification, 0))
		})

		t.Run("Test fetch notifications limit 0", func(t *testing.T) {
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now-database.getDelayedTimeInSeconds(), 0) //nolint
			require.Error(t, err)
			require.Nil(t, actual) //nolint
		})

		t.Run("Test remove notifications by key", func(t *testing.T) {
			id1 := "id1"
			notification1 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
			}
			notification2 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
			}
			notification3 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + database.getDelayedTimeInSeconds(),
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
			}
			addNotifications(t, database, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification1, &notification2, &notification3}, actual)

			total, err = database.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now), id1, id1}, ""))
			require.NoError(t, err)
			require.EqualValues(t, 2, total)

			actual, total, err = database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification3}, actual)

			total, err = database.RemoveNotification(strings.Join([]string{fmt.Sprintf("%v", now+database.getDelayedTimeInSeconds()), id1, id1}, "")) //nolint
			require.NoError(t, err)
			require.EqualValues(t, 1, total)

			actual, total, err = database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 0, total)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)

			actual, err = database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)
		})

		t.Run("Test remove all notifications", func(t *testing.T) {
			id1 := "id1"
			notification1 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  1,
				Timestamp: now,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
			}
			notification2 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  2,
				Timestamp: now,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
			}
			notification3 := moira.ScheduledNotification{
				Contact:   moira.ContactData{ID: id1},
				Event:     moira.NotificationEvent{SubscriptionID: &id1},
				SendFail:  3,
				Timestamp: now + database.getDelayedTimeInSeconds(),
				Trigger:   moira.TriggerData{TriggerSource: moira.PrometheusRemote, ClusterId: moira.DefaultCluster},
			}
			addNotifications(t, database, []moira.ScheduledNotification{notification1, notification2, notification3})
			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 3, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification1, &notification2, &notification3}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)

			actual, total, err = database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 0, total)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)

			actual, err = database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)
		})
	})
}

func addNotifications(t *testing.T, database *DbConnector, notifications []moira.ScheduledNotification) {
	for _, notification := range notifications {
		err := database.AddNotification(&notification)
		require.NoError(t, err)
	}
}

func TestScheduledNotificationErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabaseWithIncorrectConfig(logger)
	database.Flush()

	defer database.Flush()

	t.Run("Should throw error when no connection", func(t *testing.T) {
		actual1, total, err := database.GetNotifications(0, 1)
		require.Nil(t, actual1)
		require.EqualValues(t, 0, total)
		require.Error(t, err)

		total, err = database.RemoveNotification("123")
		require.Error(t, err)
		require.EqualValues(t, 0, total)

		actual2, err := database.FetchNotifications(moira.DefaultLocalCluster, 0, notificationsLimitUnlimited)
		require.Error(t, err)
		require.Nil(t, actual2)

		notification := moira.ScheduledNotification{}
		err = database.AddNotification(&notification)
		require.Error(t, err)

		err = database.AddNotifications([]*moira.ScheduledNotification{&notification}, 0)
		require.Error(t, err)

		err = database.RemoveAllNotifications()
		require.Error(t, err)
	})
}

func TestFetchNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	t.Run("FetchNotifications manipulation", func(t *testing.T) {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test fetch notifications with limit if all notifications has diff timestamp", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), 1) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)

			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 2, total)
			require.Equal(t, []*moira.ScheduledNotification{&notification, &notificationNew}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test fetch notifications with limit little bit greater than count if all notifications has diff timestamp", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), 4) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification}, actual)

			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Equal(t, []*moira.ScheduledNotification{&notificationNew}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test fetch notifications with limit greater than count if all notifications has diff timestamp", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), 200000) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)
			require.EqualValues(t, 0, total)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test fetch notifications without limit", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.FetchNotifications(moira.DefaultLocalCluster, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

			actual, total, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, actual)
			require.EqualValues(t, 0, total)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})
	})
}

func TestGetNotificationsInTxWithLimit(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	redisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

	client := *database.client
	ctx := database.context

	t.Run("Test getNotificationsInTxWithLimit", func(t *testing.T) {
		var limit int64 = 0

		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test with zero notifications without limit", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{})

			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(redisKey, ctx, tx, now+database.getDelayedTimeInSeconds()*2, notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications without limit", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(redisKey, ctx, tx, now+database.getDelayedTimeInSeconds()*2, notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications with limit != count", func(t *testing.T) {
			limit = 1

			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(redisKey, ctx, tx, now+database.getDelayedTimeInSeconds()*2, limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications with limit = count", func(t *testing.T) {
			limit = 3

			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getNotificationsInTxWithLimit(redisKey, ctx, tx, now+database.getDelayedTimeInSeconds()*2, limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})
	})
}

func TestGetLimitedNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	redisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

	client := *database.client
	ctx := database.context

	t.Run("Test getLimitedNotifications", func(t *testing.T) {
		var limit int64

		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test all notifications with different timestamps without limit", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(redisKey, ctx, tx, notificationsLimitUnlimited, notifications)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications with different timestamps and limit", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(redisKey, ctx, tx, limit, notifications)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification}, actual)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications with same timestamp and limit", func(t *testing.T) {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now

			defer func() {
				notificationNew.Timestamp = now + database.getDelayedTimeInSeconds()
				notificationOld.Timestamp = now - database.getDelayedTimeInSeconds()
			}()

			addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(redisKey, ctx, tx, limit, notifications)
				require.NoError(t, err)
				require.ElementsMatch(t, actual, expected)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test not all notifications with same timestamp and limit", func(t *testing.T) {
			notification.Timestamp = now
			notificationNew.Timestamp = now
			notificationOld.Timestamp = now

			defer func() {
				notificationNew.Timestamp = now + database.getDelayedTimeInSeconds()
				notificationOld.Timestamp = now - database.getDelayedTimeInSeconds()
			}()

			addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notification, notificationNew})
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification}
			expected := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			err := client.Watch(ctx, func(tx *redis.Tx) error {
				actual, err := getLimitedNotifications(redisKey, ctx, tx, limit, notifications)
				require.NoError(t, err)
				require.ElementsMatch(t, actual, expected)

				return nil
			}, notifierNotificationsKey)
			require.NoError(t, err)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
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

	t.Run("Test filter notifications by state", func(t *testing.T) {
		t.Run("With empty notifications", func(t *testing.T) {
			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToRemove)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveOld)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveNew)
		})

		t.Run("With all valid notifications", func(t *testing.T) {
			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification, notificationNew}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToRemove)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveOld)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveNew)
		})

		t.Run("With removed check data", func(t *testing.T) {
			database.RemoveTriggerLastCheck("test1") //nolint

			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)
			}()

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationNew}, types.ToRemove)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveOld)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveNew)
		})

		t.Run("With metric on maintenance", func(t *testing.T) {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notification, notificationNew}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToRemove)
			require.Len(t, types.ToResaveOld, 1)
			require.Equal(t, types.ToResaveOld[0], notificationOld)
			require.Len(t, types.ToResaveNew, 1)
			require.Equal(t, types.ToResaveNew[0].SendFail, notificationOld.SendFail)
		})

		t.Run("With trigger on maintenance", func(t *testing.T) {
			triggerMaintenance := time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint

			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			types, err := database.filterNotificationsByState([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToRemove)
			require.Len(t, types.ToResaveOld, 1)
			require.Equal(t, types.ToResaveOld[0], notificationNew)
			require.Len(t, types.ToResaveNew, 1)
			require.Equal(t, types.ToResaveNew[0].SendFail, notificationNew.SendFail)
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

	t.Run("Test handle notifications", func(t *testing.T) {
		t.Run("Without delayed notifications", func(t *testing.T) {
			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notification, notificationNew})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification, notificationNew}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification, notificationNew}, types.ToRemove)

			var toResaveNotificationsExpected []*moira.ScheduledNotification

			require.Equal(t, toResaveNotificationsExpected, types.ToResaveOld)
			require.Equal(t, toResaveNotificationsExpected, types.ToResaveNew)
		})

		t.Run("With both delayed and not delayed valid notifications", func(t *testing.T) {
			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3}, types.ToRemove)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveOld)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveNew)
		})

		t.Run("With both delayed and not delayed notifications and removed check data", func(t *testing.T) {
			database.RemoveTriggerLastCheck("test1") //nolint

			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{}, defaultSourceNotSetCluster)
			}()

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationNew2, notificationOld, notificationOld2, notification, notificationNew, notificationNew3}, types.ToRemove)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveOld)
			require.Equal(t, []*moira.ScheduledNotification{}, types.ToResaveNew)
		})

		t.Run("With both delayed and not delayed valid notifications and metric on maintenance", func(t *testing.T) {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test": 0}, nil, "test", 100)                          //nolint

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification, notificationNew, notificationNew2, notificationNew3}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notification, notificationNew, notificationNew2, notificationNew3}, types.ToRemove)
			require.Len(t, types.ToResaveNew, 1)
			require.Equal(t, types.ToResaveNew[0].SendFail, notificationOld2.SendFail)
			require.Len(t, types.ToResaveOld, 1)
			require.Equal(t, types.ToResaveOld[0], notificationOld2)
		})

		t.Run("With both delayed and not delayed valid notifications and trigger on maintenance", func(t *testing.T) {
			triggerMaintenance := time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint

			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			types, err := database.handleNotifications([]*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3}, types.Valid)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew3}, types.ToRemove)
			require.Len(t, types.ToResaveNew, 1)
			require.Equal(t, types.ToResaveNew[0].SendFail, notificationNew2.SendFail)
			require.Len(t, types.ToResaveOld, 1)
			require.Equal(t, types.ToResaveOld[0], notificationNew2)
		})
	})
}

func TestNotificationsCount(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	redisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

	t.Run("notificationsCount in db", func(t *testing.T) {
		now = time.Now().Unix()
		notificationNew := moira.ScheduledNotification{
			SendFail:  1,
			Timestamp: now + database.getDelayedTimeInSeconds(),
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notification := moira.ScheduledNotification{
			SendFail:  2,
			Timestamp: now,
			CreatedAt: now,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld := moira.ScheduledNotification{
			SendFail:  3,
			Timestamp: now - database.getDelayedTimeInSeconds(),
			CreatedAt: now - database.getDelayedTimeInSeconds(),
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test all notification with different ts in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(redisKey, now+database.getDelayedTimeInSeconds()*2)
			require.NoError(t, err)
			require.Equal(t, int64(3), actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test get 0 notification with ts in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(redisKey, now-database.getDelayedTimeInSeconds()*2)
			require.NoError(t, err)
			require.Equal(t, int64(0), actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test part notification in db with ts", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
			actual, err := database.notificationsCount(redisKey, now)
			require.NoError(t, err)
			require.Equal(t, int64(2), actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test 0 notification in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{})
			actual, err := database.notificationsCount(redisKey, now)
			require.NoError(t, err)
			require.Equal(t, int64(0), actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test different redis keys", func(t *testing.T) {
			notificationRemote := moira.ScheduledNotification{
				SendFail:  1,
				Timestamp: now - database.getDelayedTimeInSeconds(),
				CreatedAt: now,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
			}
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationOld, notificationRemote})

			actual, err := database.notificationsCount(redisKey, now)
			require.NoError(t, err)
			require.Equal(t, int64(2), actual)

			actual, err = database.notificationsCount(makeNotifierNotificationsKey(moira.DefaultGraphiteRemoteCluster), now)
			require.NoError(t, err)
			require.Equal(t, int64(1), actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
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
	redisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

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
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notification4 := moira.ScheduledNotification{
		SendFail:  2,
		Timestamp: now - database.getDelayedTimeInSeconds() + 2,
		CreatedAt: now - database.getDelayedTimeInSeconds() + 2,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notificationNew := moira.ScheduledNotification{
		SendFail:  3,
		Timestamp: now + database.getDelayedTimeInSeconds(),
		CreatedAt: now,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notification := moira.ScheduledNotification{
		SendFail:  4,
		Timestamp: now,
		CreatedAt: now,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}

	// create delayed notifications
	notificationOld2 := moira.ScheduledNotification{
		Trigger: moira.TriggerData{
			ID:            "test2",
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
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
			ID:            "test1",
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
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
			ID:            "test2",
			TriggerSource: moira.GraphiteLocal,
			ClusterId:     moira.DefaultCluster,
		},
		Event: moira.NotificationEvent{
			Metric: "test2",
		},
		SendFail:  7,
		Timestamp: now + database.getDelayedTimeInSeconds() + 2,
		CreatedAt: now,
	}

	t.Run("Test fetchNotificationsDo", func(t *testing.T) {
		t.Run("Test all notifications with diff ts in db", func(t *testing.T) {
			t.Run("With limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

				limit = 1
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})

			t.Run("Without limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}, actual)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})
		})

		t.Run("Test zero notifications with ts in empty db", func(t *testing.T) {
			t.Run("With limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{})

				limit = 10
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{}, actual)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})

			t.Run("Without limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{})
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{}, actual)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})
		})

		t.Run("Test all notification with ts and without limit in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})
			actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), notificationsLimitUnlimited)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification4, &notification, &notificationNew}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test all notifications with ts and big limit in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

			limit = 100
			actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), limit) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test notifications with ts and small limit in db", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld, notification4})

			limit = 3
			actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), limit) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification4}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test notifications with ts and limit = count", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{notification, notificationNew, notificationOld})

			limit = 3
			actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds(), limit) //nolint
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification}, actual)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test delayed notifications with ts and deleted trigger", func(t *testing.T) {
			database.RemoveTriggerLastCheck("test1") //nolint

			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {},
					},
				}, defaultSourceNotSetCluster)
			}()

			t.Run("With big limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})

				limit = 100
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationNew3}, allNotifications)
				require.EqualValues(t, 1, count)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})

			t.Run("Without limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{}, allNotifications)
				require.EqualValues(t, 0, count)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})
		})

		t.Run("Test notifications with ts and metric on maintenance", func(t *testing.T) {
			database.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": time.Now().Add(time.Hour).Unix()}, nil, "test", 100) //nolint
			defer database.SetTriggerCheckMaintenance("test2", map[string]int64{"test2": 0}, nil, "test", 100)                          //nolint

			t.Run("With limit = count", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})

				limit = 6
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationNew3}, allNotifications)
				require.EqualValues(t, 1, count)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})

			t.Run("Without limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew2}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
				require.Equal(t, allNotifications[0].SendFail, notificationNew3.SendFail)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})
		})

		t.Run("Test delayed notifications with ts and trigger on maintenance", func(t *testing.T) {
			triggerMaintenance := time.Now().Add(time.Hour).Unix()
			database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint

			defer func() {
				triggerMaintenance = 0
				database.SetTriggerCheckMaintenance("test1", map[string]int64{}, &triggerMaintenance, "test", 100) //nolint
			}()

			t.Run("With small limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})

				limit = 3
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, limit)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notification, &notificationNew, &notificationNew2, &notificationNew3}, allNotifications)
				require.EqualValues(t, 4, count)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})

			t.Run("without limit", func(t *testing.T) {
				addNotifications(t, database, []moira.ScheduledNotification{notificationOld, notificationOld2, notification, notificationNew, notificationNew2, notificationNew3})
				actual, err := database.fetchNotificationsTx(redisKey, now+database.getDelayedTimeInSeconds()+3, notificationsLimitUnlimited)
				require.NoError(t, err)
				require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notificationOld2, &notification, &notificationNew, &notificationNew3}, actual)

				allNotifications, count, err := database.GetNotifications(0, -1)
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
				require.Equal(t, allNotifications[0].SendFail, notificationNew2.SendFail)

				err = database.RemoveAllNotifications()
				require.NoError(t, err)
			})
		})
	})
}

func TestLimitNotifications(t *testing.T) { //nolint
	t.Run("limitNotifications manipulation", func(t *testing.T) {
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

		t.Run("Test limit Notifications zero notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{}
			actual := limitNotifications(notifications)
			require.Equal(t, actual, notifications)
		})

		t.Run("Test limit Notifications notifications diff ts", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification, &notificationNew}
			actual := limitNotifications(notifications)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification}, actual)
		})

		t.Run("Test limit Notifications all notifications same ts", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification4}
			actual := limitNotifications(notifications)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification4}, actual)
		})

		t.Run("Test limit Notifications 1 notifications diff ts and 2 same ts", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notificationNew, &notification5}
			actual := limitNotifications(notifications)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld}, actual)
		})

		t.Run("Test limit Notifications 2 notifications same ts and 1 diff ts", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{&notificationOld, &notification4, &notification5}
			actual := limitNotifications(notifications)
			require.Equal(t, []*moira.ScheduledNotification{&notificationOld, &notification4}, actual)
		})
	})
}

func TestFilterNotificationsByDelay(t *testing.T) {
	t.Run("filterNotificationsByDelay manipulations", func(t *testing.T) {
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

		t.Run("Test with zero notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 1)
			require.Equal(t, []*moira.ScheduledNotification{}, delayed)
			require.Equal(t, []*moira.ScheduledNotification{}, notDelayed)
		})

		t.Run("Test with zero delayed notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 50)
			require.Equal(t, []*moira.ScheduledNotification{}, delayed)
			require.Equal(t, notDelayed, notifications)
		})

		t.Run("Test with zero not delayed notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 2)
			require.Equal(t, delayed, notifications)
			require.Equal(t, []*moira.ScheduledNotification{}, notDelayed)
		})

		t.Run("Test with one delayed and two not delayed notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			delayed, notDelayed := filterNotificationsByDelay(notifications, 15)
			require.Equal(t, []*moira.ScheduledNotification{notification3}, delayed)
			require.Equal(t, []*moira.ScheduledNotification{notification1, notification2}, notDelayed)
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

	t.Run("getNotificationsTriggerChecks manipulations", func(t *testing.T) {
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

		t.Run("Test with zero notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			require.NoError(t, err)
			require.Equal(t, []*moira.CheckData{}, triggerChecks)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test with correct notifications", func(t *testing.T) {
			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			require.NoError(t, err)
			require.Equal(t, []*moira.CheckData{
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
			}, triggerChecks)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test notifications with removed test1 trigger check", func(t *testing.T) {
			database.RemoveTriggerLastCheck("test1") //nolint

			defer func() {
				_ = database.SetTriggerLastCheck("test1", &moira.CheckData{
					Timestamp: 1,
				}, defaultSourceNotSetCluster)
			}()

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			require.NoError(t, err)
			require.Equal(t, []*moira.CheckData{nil, nil, {Timestamp: 2, MetricsToTargetRelation: map[string]string{}, Clock: clock.NewSystemClock()}}, triggerChecks)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test notifications with removed all trigger checks", func(t *testing.T) {
			database.RemoveTriggerLastCheck("test1") //nolint
			database.RemoveTriggerLastCheck("test2") //nolint

			notifications := []*moira.ScheduledNotification{notification1, notification2, notification3}
			triggerChecks, err := database.getNotificationsTriggerChecks(notifications)
			require.NoError(t, err)
			require.Equal(t, []*moira.CheckData{nil, nil, nil}, triggerChecks)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})
	})
}

func TestResaveNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	redisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

	client := database.client
	ctx := database.context
	pipe := (*client).TxPipeline()

	t.Run("Test resaveNotifications", func(t *testing.T) {
		notificationOld1 := &moira.ScheduledNotification{
			Timestamp: 1,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld2 := &moira.ScheduledNotification{
			Timestamp: 2,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationOld3 := &moira.ScheduledNotification{
			Timestamp: 3,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		notificationNew1 := &moira.ScheduledNotification{
			Timestamp: 4,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationNew2 := &moira.ScheduledNotification{
			Timestamp: 5,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}
		notificationNew3 := &moira.ScheduledNotification{
			Timestamp: 6,
			Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
		}

		t.Run("Test resave with zero notifications", func(t *testing.T) {
			affected, err := database.resaveNotifications(redisKey, ctx, pipe, []*moira.ScheduledNotification{}, []*moira.ScheduledNotification{})
			require.NoError(t, err)
			require.Equal(t, 0, affected)

			allNotifications, count, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{}, allNotifications)
			require.EqualValues(t, 0, count)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test resave notifications with empty database", func(t *testing.T) {
			affected, err := database.resaveNotifications(redisKey, ctx, pipe, []*moira.ScheduledNotification{notificationOld1, notificationOld2}, []*moira.ScheduledNotification{notificationNew1, notificationNew2})
			require.NoError(t, err)
			require.Equal(t, 2, affected)

			allNotifications, count, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationNew1, notificationNew2}, allNotifications)
			require.EqualValues(t, 2, count)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test resave one notification when other notifications exist in the database", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{*notificationOld1, *notificationOld3})

			affected, err := database.resaveNotifications(redisKey, ctx, pipe, []*moira.ScheduledNotification{notificationOld2}, []*moira.ScheduledNotification{notificationNew2})
			require.NoError(t, err)
			require.Equal(t, 1, affected)

			allNotifications, count, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationOld1, notificationOld3, notificationNew2}, allNotifications)
			require.EqualValues(t, 3, count)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test resave all notifications", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{*notificationOld1, *notificationOld2, *notificationOld3})

			affected, err := database.resaveNotifications(redisKey, ctx, pipe, []*moira.ScheduledNotification{notificationOld1, notificationOld2, notificationOld3}, []*moira.ScheduledNotification{notificationNew1, notificationNew2, notificationNew3})
			require.NoError(t, err)
			require.Equal(t, 6, affected)

			allNotifications, count, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationNew1, notificationNew2, notificationNew3}, allNotifications)
			require.EqualValues(t, 3, count)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})

		t.Run("Test resave notifications with different redis keys", func(t *testing.T) {
			notificationOld4 := &moira.ScheduledNotification{
				Timestamp: 7,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
			}

			notificationNew4 := &moira.ScheduledNotification{
				Timestamp: 8,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
			}

			redisKey := makeNotifierNotificationsKey(moira.DefaultGraphiteRemoteCluster)

			addNotifications(t, database, []moira.ScheduledNotification{*notificationOld4})

			affected, err := database.resaveNotifications(redisKey, ctx, pipe, []*moira.ScheduledNotification{notificationOld4}, []*moira.ScheduledNotification{notificationNew4})
			require.NoError(t, err)
			require.Equal(t, 2, affected)

			allNotifications, count, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.Equal(t, []*moira.ScheduledNotification{notificationNew4}, allNotifications)
			require.EqualValues(t, 1, count)

			err = database.RemoveAllNotifications()
			require.NoError(t, err)
		})
	})
}

func TestRemoveNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	localDefaultRedisKey := makeNotifierNotificationsKey(moira.DefaultLocalCluster)

	client := database.client
	ctx := database.context
	pipe := (*client).TxPipeline()

	notification1 := &moira.ScheduledNotification{
		Timestamp: 1,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notification2 := &moira.ScheduledNotification{
		Timestamp: 2,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notification3 := &moira.ScheduledNotification{
		Timestamp: 3,
		Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}

	t.Run("Test removeNotifications", func(t *testing.T) {
		t.Run("Test remove empty notifications", func(t *testing.T) {
			count, err := database.removeNotifications(localDefaultRedisKey, ctx, pipe, []*moira.ScheduledNotification{})
			require.NoError(t, err)
			require.EqualValues(t, 0, count)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 0, countAllNotifications)
			require.Equal(t, []*moira.ScheduledNotification{}, allNotifications)
		})

		t.Run("Test remove one notification", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(localDefaultRedisKey, ctx, pipe, []*moira.ScheduledNotification{notification2})
			require.NoError(t, err)
			require.EqualValues(t, 1, count)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 2, countAllNotifications)
			require.Equal(t, []*moira.ScheduledNotification{notification1, notification3}, allNotifications)
		})

		t.Run("Test remove all notifications", func(t *testing.T) {
			addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(localDefaultRedisKey, ctx, pipe, []*moira.ScheduledNotification{notification1, notification2, notification3})
			require.NoError(t, err)
			require.EqualValues(t, 3, count)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 0, countAllNotifications)
			require.Equal(t, []*moira.ScheduledNotification{}, allNotifications)
		})

		t.Run("Test remove a nonexistent notification", func(t *testing.T) {
			notification4 := &moira.ScheduledNotification{
				Timestamp: 4,
			}

			addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3})

			count, err := database.removeNotifications(localDefaultRedisKey, ctx, pipe, []*moira.ScheduledNotification{notification4})
			require.NoError(t, err)
			require.EqualValues(t, 0, count)

			allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
			require.NoError(t, err)
			require.EqualValues(t, 3, countAllNotifications)
			require.Equal(t, []*moira.ScheduledNotification{notification1, notification2, notification3}, allNotifications)
		})

		t.Run("Test remove notifications from different sources", func(t *testing.T) {
			notification4 := &moira.ScheduledNotification{
				Timestamp: 2,
				Trigger:   moira.TriggerData{TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
			}
			notification5 := &moira.ScheduledNotification{
				Timestamp: 3,
				Trigger:   moira.TriggerData{TriggerSource: moira.PrometheusRemote, ClusterId: moira.DefaultCluster},
			}

			addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification4, *notification5})

			count, err := database.removeNotifications(localDefaultRedisKey, ctx, pipe, []*moira.ScheduledNotification{notification1})
			require.NoError(t, err)
			require.EqualValues(t, 1, count)

			count, err = database.removeNotifications(makeNotifierNotificationsKey(moira.DefaultGraphiteRemoteCluster), ctx, pipe, []*moira.ScheduledNotification{notification4})
			require.NoError(t, err)
			require.EqualValues(t, 1, count)

			count, err = database.removeNotifications(makeNotifierNotificationsKey(moira.DefaultPrometheusRemoteCluster), ctx, pipe, []*moira.ScheduledNotification{notification5})
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
		})
	})
}

func TestRemoveFilteredNotifications(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	database := NewTestDatabase(logger)
	database.Flush()

	defer database.Flush()

	ignoredTag := "ignored"
	tag := "tag"

	notification1 := &moira.ScheduledNotification{
		Timestamp: 10,
		Trigger:   moira.TriggerData{Tags: []string{tag}, TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}
	notification2 := &moira.ScheduledNotification{
		Timestamp: 20,
		Trigger:   moira.TriggerData{Tags: []string{tag, ignoredTag}, TriggerSource: moira.PrometheusRemote, ClusterId: moira.DefaultCluster},
	}
	notification3 := &moira.ScheduledNotification{
		Timestamp: 30,
		Trigger:   moira.TriggerData{Tags: []string{ignoredTag}, TriggerSource: moira.GraphiteRemote, ClusterId: moira.DefaultCluster},
	}
	notification4 := &moira.ScheduledNotification{
		Timestamp: 40,
		Trigger:   moira.TriggerData{Tags: []string{}, TriggerSource: moira.GraphiteLocal, ClusterId: moira.DefaultCluster},
	}

	t.Run("Test removeNotifications filtered for all time", func(t *testing.T) {
		addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3, *notification4})

		count, err := database.RemoveFilteredNotifications(0, -1, []string{ignoredTag}, []moira.ClusterKey{})
		require.NoError(t, err)
		require.EqualValues(t, 2, count)

		allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
		require.NoError(t, err)
		require.EqualValues(t, 2, countAllNotifications)
		require.Equal(t, []*moira.ScheduledNotification{notification2, notification3}, allNotifications)
	})

	t.Run("Test removeNotifications filtered for time range", func(t *testing.T) {
		addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3, *notification4})

		count, err := database.RemoveFilteredNotifications(25, 45, []string{ignoredTag}, []moira.ClusterKey{})
		require.NoError(t, err)
		require.EqualValues(t, 1, count)

		allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
		require.NoError(t, err)
		require.EqualValues(t, 3, countAllNotifications)

		require.Equal(t, []*moira.ScheduledNotification{notification1, notification2, notification3}, allNotifications)
	})

	database.Flush()

	t.Run("Test removeNotifications filtered by cluster key", func(t *testing.T) {
		addNotifications(t, database, []moira.ScheduledNotification{*notification1, *notification2, *notification3, *notification4})

		count, err := database.RemoveFilteredNotifications(0, -1, []string{}, []moira.ClusterKey{moira.DefaultLocalCluster})
		require.NoError(t, err)
		require.EqualValues(t, 2, count)

		allNotifications, countAllNotifications, err := database.GetNotifications(0, -1)
		require.NoError(t, err)
		require.EqualValues(t, 2, countAllNotifications)

		require.Equal(t, []*moira.ScheduledNotification{notification2, notification3}, allNotifications)
	})
}
