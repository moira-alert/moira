package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

var testTimestamp = time.Now().Unix()

const (
	defaultTestMetric = "test.notification.events"
)

var testNotificationHistoryEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testTimestamp - 1,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-4",
	},
	{
		TimeStamp: testTimestamp,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-1",
	},
	{
		TimeStamp: testTimestamp + 1,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testTimestamp + 2,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
	{
		TimeStamp: testTimestamp + 3,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
	{
		TimeStamp: testTimestamp + 4,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testTimestamp + 5,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
}

var additionalTestNotificationHistoryEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testTimestamp + 6,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-4",
	},
	{
		TimeStamp: testTimestamp + 7,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-5",
	},
}

func TestSplitNotificationHistory(t *testing.T) {
	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	require.NoError(t, err)

	db := redis.NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	t.Run("Test split notification history", func(t *testing.T) {
		t.Run("with empty contactNotificationKey", func(t *testing.T) {
			err = splitNotificationHistoryByContactID(ctx, logger, db)
			require.NoError(t, err)

			keys, err := client.Keys(ctx, contactNotificationKeyWithID("*")).Result()
			require.NoError(t, err)
			require.Len(t, keys, 0)
		})

		t.Run("with empty split history", func(t *testing.T) {
			toInsert, err := prepareNotSplitItemsToInsert(testNotificationHistoryEvents)
			require.NoError(t, err)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			require.NoError(t, err)

			eventsMap := eventsByKey(testNotificationHistoryEvents)

			testSplitNotificationHistory(t, ctx, db, logger, eventsMap)

			db.Flush()
		})

		t.Run("with not empty split history", func(t *testing.T) {
			toInsert, err := prepareNotSplitItemsToInsert(testNotificationHistoryEvents)
			require.NoError(t, err)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			require.NoError(t, err)

			toInsertMap, err := prepareSplitItemsToInsert(eventsByKey(additionalTestNotificationHistoryEvents))
			require.NoError(t, err)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			require.NoError(t, err)

			testSplitNotificationHistory(
				t, ctx, db, logger, eventsByKey(append(testNotificationHistoryEvents, additionalTestNotificationHistoryEvents...)))

			db.Flush()
		})
	})
}

func testSplitNotificationHistory(
	t *testing.T,
	ctx context.Context,
	db *redis.DbConnector,
	logger moira.Logger,
	eventsMap map[string][]*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	errExists := splitNotificationHistoryByContactID(ctx, logger, db)
	require.NoError(t, errExists)

	for contactID, expectedEvents := range eventsMap {
		t.Run(fmt.Sprintf("check contact with id: %s", contactID), func(t *testing.T) {
			gotEvents, errAfterZRange := client.ZRangeByScore(
				ctx,
				contactNotificationKeyWithID(contactID),
				&goredis.ZRangeBy{
					Min:    "-inf",
					Max:    "+inf",
					Offset: 0,
					Count:  -1,
				}).Result()
			require.NoError(t, errAfterZRange)
			require.Len(t, gotEvents, len(expectedEvents))

			for i, gotEventStr := range gotEvents {
				notificationEvent, err := redis.GetNotificationStruct(gotEventStr)
				require.NoError(t, err)
				require.Equal(t, *expectedEvents[i], notificationEvent)
			}
		})
	}

	res, errExists := client.Exists(ctx, contactNotificationKey).Result()
	require.NoError(t, errExists)
	require.Equal(t, int64(0), res)
}

func TestMergeNotificationHistory(t *testing.T) {
	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	require.NoError(t, err)

	db := redis.NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	t.Run("Test merge notification history", func(t *testing.T) {
		t.Run("with empty database", func(t *testing.T) {
			err = mergeNotificationHistory(logger, db)
			require.NoError(t, err)

			keys, err := client.Keys(ctx, contactNotificationKey).Result()
			require.NoError(t, err)
			require.Len(t, keys, 0)
		})

		t.Run("with empty history by single key", func(t *testing.T) {
			eventsMap := eventsByKey(testNotificationHistoryEvents)

			toInsertMap, err := prepareSplitItemsToInsert(eventsMap)
			require.NoError(t, err)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			require.NoError(t, err)

			testMergeNotificationHistory(t, ctx, db, logger, testNotificationHistoryEvents)

			db.Flush()
		})

		t.Run("with not empty history by single key", func(t *testing.T) {
			eventsMap := eventsByKey(testNotificationHistoryEvents)

			toInsertMap, err := prepareSplitItemsToInsert(eventsMap)
			require.NoError(t, err)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			require.NoError(t, err)

			toInsert, err := prepareNotSplitItemsToInsert(additionalTestNotificationHistoryEvents)
			require.NoError(t, err)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			require.NoError(t, err)

			testMergeNotificationHistory(
				t, ctx, db, logger, append(testNotificationHistoryEvents, additionalTestNotificationHistoryEvents...))

			db.Flush()
		})
	})
}

func prepareNotSplitItemsToInsert(notificationEvents []*moira.NotificationEventHistoryItem) ([]*goredis.Z, error) {
	resList := make([]*goredis.Z, 0, len(notificationEvents))

	for _, notificationEvent := range notificationEvents {
		toInsert, err := toInsertableItem(notificationEvent)
		if err != nil {
			return nil, err
		}

		resList = append(resList, toInsert)
	}

	return resList, nil
}

func toInsertableItem(notificationEvent *moira.NotificationEventHistoryItem) (*goredis.Z, error) {
	notificationBytes, err := redis.GetNotificationBytes(notificationEvent)
	if err != nil {
		return nil, err
	}

	return &goredis.Z{Score: float64(notificationEvent.TimeStamp), Member: notificationBytes}, nil
}

func storeNotificationHistoryBySingleKey(ctx context.Context, database moira.Database, toInsert []*goredis.Z) error {
	switch db := database.(type) {
	case *redis.DbConnector:
		client := db.Client()

		_, err := client.ZAdd(ctx, contactNotificationKey, toInsert...).Result()
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	return nil
}

func eventsByKey(notificationEvents []*moira.NotificationEventHistoryItem) map[string][]*moira.NotificationEventHistoryItem {
	statistics := make(map[string][]*moira.NotificationEventHistoryItem, len(notificationEvents))
	for _, event := range notificationEvents {
		statistics[event.ContactID] = append(statistics[event.ContactID], event)
	}

	return statistics
}

func prepareSplitItemsToInsert(eventsMap map[string][]*moira.NotificationEventHistoryItem) (map[string][]*goredis.Z, error) {
	resMap := make(map[string][]*goredis.Z, len(eventsMap))

	for contactID, notificationEvents := range eventsMap {
		for _, notificationEvent := range notificationEvents {
			toInsert, err := toInsertableItem(notificationEvent)
			if err != nil {
				return nil, err
			}

			resMap[contactID] = append(resMap[contactID], toInsert)
		}
	}

	return resMap, nil
}

func storeSplitNotifications(ctx context.Context, database moira.Database, toInsertMap map[string][]*goredis.Z) error {
	switch db := database.(type) {
	case *redis.DbConnector:
		client := db.Client()

		pipe := client.TxPipeline()

		for contactID, insertItems := range toInsertMap {
			key := contactNotificationKeyWithID(contactID)
			for _, z := range insertItems {
				pipe.ZAdd(ctx, key, z)
			}
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
	default:
		return makeUnknownDBError(database)
	}

	return nil
}

func testMergeNotificationHistory(
	t *testing.T,
	ctx context.Context,
	db *redis.DbConnector,
	logger moira.Logger,
	eventsList []*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	err := mergeNotificationHistory(logger, db)
	require.NoError(t, err)

	gotEventsStrs, errAfterZRange := client.ZRangeByScore(
		ctx,
		contactNotificationKey,
		&goredis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: 0,
			Count:  -1,
		}).Result()
	require.NoError(t, errAfterZRange)
	require.Len(t, gotEventsStrs, len(eventsList))

	for i, gotEventStr := range gotEventsStrs {
		notificationEvent, errDeserialize := redis.GetNotificationStruct(gotEventStr)
		require.NoError(t, errDeserialize)
		require.Equal(t, *eventsList[i], notificationEvent)
	}

	contactKeys, errAfterKeys := client.Keys(ctx, contactNotificationKeyWithID("*")).Result()
	require.NoError(t, errAfterKeys)
	require.Len(t, contactKeys, 0)
}
