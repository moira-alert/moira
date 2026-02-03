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
	. "github.com/smartystreets/goconvey/convey"
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
	if err != nil {
		t.Fatal(err)
	}

	db := redis.NewTestDatabase(logger)
	db.Flush()

	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	Convey("Test split notification history", t, func() {
		Convey("with empty contactNotificationKey", func() {
			err = splitNotificationHistoryByContactID(ctx, logger, db)
			So(err, ShouldBeNil)

			keys, err := client.Keys(ctx, contactNotificationKeyWithID("*")).Result()
			So(err, ShouldBeNil)
			So(keys, ShouldHaveLength, 0)
		})

		Convey("with empty split history", func() {
			toInsert, err := prepareNotSplitItemsToInsert(testNotificationHistoryEvents)
			So(err, ShouldBeNil)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			So(err, ShouldBeNil)

			eventsMap := eventsByKey(testNotificationHistoryEvents)

			testSplitNotificationHistory(ctx, db, logger, eventsMap)

			db.Flush()
		})

		Convey("with not empty split history", func() {
			toInsert, err := prepareNotSplitItemsToInsert(testNotificationHistoryEvents)
			So(err, ShouldBeNil)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			So(err, ShouldBeNil)

			toInsertMap, err := prepareSplitItemsToInsert(eventsByKey(additionalTestNotificationHistoryEvents))
			So(err, ShouldBeNil)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			So(err, ShouldBeNil)

			testSplitNotificationHistory(
				ctx, db, logger, eventsByKey(append(testNotificationHistoryEvents, additionalTestNotificationHistoryEvents...)))

			db.Flush()
		})
	})
}

func testSplitNotificationHistory(
	ctx context.Context,
	db *redis.DbConnector,
	logger moira.Logger,
	eventsMap map[string][]*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	Convey("with prepared history", func() {
		errExists := splitNotificationHistoryByContactID(ctx, logger, db)
		So(errExists, ShouldBeNil)

		for contactID, expectedEvents := range eventsMap {
			Convey(fmt.Sprintf("check contact with id: %s", contactID), func() {
				gotEvents, errAfterZRange := client.ZRangeByScore(
					ctx,
					contactNotificationKeyWithID(contactID),
					&goredis.ZRangeBy{
						Min:    "-inf",
						Max:    "+inf",
						Offset: 0,
						Count:  -1,
					}).Result()
				So(errAfterZRange, ShouldBeNil)
				So(gotEvents, ShouldHaveLength, len(expectedEvents))

				for i, gotEventStr := range gotEvents {
					notificationEvent, err := redis.GetNotificationStruct(gotEventStr)
					So(err, ShouldBeNil)
					So(notificationEvent, ShouldResemble, *expectedEvents[i])
				}
			})
		}

		res, errExists := client.Exists(ctx, contactNotificationKey).Result()
		So(errExists, ShouldBeNil)
		So(res, ShouldEqual, 0)
	})
}

func TestMergeNotificationHistory(t *testing.T) {
	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}

	db := redis.NewTestDatabase(logger)
	db.Flush()

	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	Convey("Test merge notification history", t, func() {
		Convey("with empty database", func() {
			err = mergeNotificationHistory(logger, db)
			So(err, ShouldBeNil)

			keys, err := client.Keys(ctx, contactNotificationKey).Result()
			So(err, ShouldBeNil)
			So(keys, ShouldHaveLength, 0)
		})

		Convey("with empty history by single key", func() {
			eventsMap := eventsByKey(testNotificationHistoryEvents)

			toInsertMap, err := prepareSplitItemsToInsert(eventsMap)
			So(err, ShouldBeNil)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			So(err, ShouldBeNil)

			testMergeNotificationHistory(ctx, db, logger, testNotificationHistoryEvents)

			db.Flush()
		})

		Convey("with not empty history by single key", func() {
			eventsMap := eventsByKey(testNotificationHistoryEvents)

			toInsertMap, err := prepareSplitItemsToInsert(eventsMap)
			So(err, ShouldBeNil)

			err = storeSplitNotifications(ctx, db, toInsertMap)
			So(err, ShouldBeNil)

			toInsert, err := prepareNotSplitItemsToInsert(additionalTestNotificationHistoryEvents)
			So(err, ShouldBeNil)

			err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
			So(err, ShouldBeNil)

			testMergeNotificationHistory(
				ctx, db, logger, append(testNotificationHistoryEvents, additionalTestNotificationHistoryEvents...))

			db.Flush()
		})
	})
}

func testMergeNotificationHistory(
	ctx context.Context,
	db *redis.DbConnector,
	logger moira.Logger,
	eventsList []*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	Convey("with split history", func() {
		err := mergeNotificationHistory(logger, db)
		So(err, ShouldBeNil)

		gotEventsStrs, errAfterZRange := client.ZRangeByScore(
			ctx,
			contactNotificationKey,
			&goredis.ZRangeBy{
				Min:    "-inf",
				Max:    "+inf",
				Offset: 0,
				Count:  -1,
			}).Result()
		So(errAfterZRange, ShouldBeNil)
		So(gotEventsStrs, ShouldHaveLength, len(eventsList))

		for i, gotEventStr := range gotEventsStrs {
			notificationEvent, errDeserialize := redis.GetNotificationStruct(gotEventStr)
			So(errDeserialize, ShouldBeNil)
			So(notificationEvent, ShouldResemble, *eventsList[i])
		}

		contactKeys, errAfterKeys := client.Keys(ctx, contactNotificationKeyWithID("*")).Result()
		So(errAfterKeys, ShouldBeNil)
		So(contactKeys, ShouldHaveLength, 0)
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
