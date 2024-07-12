package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	moira_redis "github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

var testTimeStamp = time.Now().Unix()

var testNotificationHistoryEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testTimeStamp - 1,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-4",
	},
	{
		TimeStamp: testTimeStamp,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-1",
	},
	{
		TimeStamp: testTimeStamp + 1,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testTimeStamp + 2,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
	{
		TimeStamp: testTimeStamp + 3,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
	{
		TimeStamp: testTimeStamp + 4,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testTimeStamp + 5,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
}

var additionalTestNotificationHistoryEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testTimeStamp + 6,
		Metric:    "test.notification.events",
		State:     "TEST",
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-4",
	},
	{
		TimeStamp: testTimeStamp + 7,
		Metric:    "test.notification.events",
		State:     "TEST",
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
	db := moira_redis.NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	Convey("Test split notification history", t, func() {
		Convey("with empty contactNotificationKey", func() {
			err = splitNotificationHistoryByContactId(ctx, logger, db, -1)
			So(err, ShouldBeNil)

			keys, err := client.Keys(ctx, contactNotificationKey+":*").Result()
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
	db *moira_redis.DbConnector,
	logger moira.Logger,
	eventsMap map[string][]*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	Convey("with prepared history", func() {
		fetchCount := int64(len(eventsMap))

		err := splitNotificationHistoryByContactId(ctx, logger, db, fetchCount)
		So(err, ShouldBeNil)

		for contactID, expectedEvents := range eventsMap {
			Convey(fmt.Sprintf("check contact with id: %s", contactID), func() {
				gotEvents, err := client.ZRangeByScore(
					ctx,
					contactNotificationKey+":"+contactID,
					&redis.ZRangeBy{
						Min:    "-inf",
						Max:    "+inf",
						Offset: 0,
						Count:  -1,
					}).Result()
				So(err, ShouldBeNil)
				So(gotEvents, ShouldHaveLength, len(expectedEvents))

				for i, gotEventStr := range gotEvents {
					notificationEvent, err := toNotificationStruct(gotEventStr)
					So(err, ShouldBeNil)
					So(notificationEvent, ShouldResemble, *expectedEvents[i])
				}
			})
		}

		res, err := client.Exists(ctx, contactNotificationKey).Result()
		So(err, ShouldBeNil)
		So(res, ShouldEqual, 0)
	})
}

func TestMergeNotificationHistory(t *testing.T) {
	conf := getDefault()
	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}
	db := moira_redis.NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	Convey("Test merge notification history", t, func() {
		Convey("with empty database", func() {
			err = mergeNotificationHistory(ctx, logger, db)
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
	db *moira_redis.DbConnector,
	logger moira.Logger,
	eventsList []*moira.NotificationEventHistoryItem,
) {
	client := db.Client()

	Convey("with split history", func() {
		err := mergeNotificationHistory(ctx, logger, db)
		So(err, ShouldBeNil)

		gotEventsStrs, err := client.ZRangeByScore(
			ctx,
			contactNotificationKey,
			&redis.ZRangeBy{
				Min:    "-inf",
				Max:    "+inf",
				Offset: 0,
				Count:  -1,
			}).Result()
		So(err, ShouldBeNil)
		So(gotEventsStrs, ShouldHaveLength, len(eventsList))

		for i, gotEventStr := range gotEventsStrs {
			notificationEvent, err := toNotificationStruct(gotEventStr)
			So(err, ShouldBeNil)
			So(notificationEvent, ShouldResemble, *eventsList[i])
		}

		contactKeys, err := client.Keys(ctx, contactNotificationKey+":*").Result()
		So(err, ShouldBeNil)
		So(contactKeys, ShouldHaveLength, 0)
	})
}

func prepareNotSplitItemsToInsert(notificationEvents []*moira.NotificationEventHistoryItem) ([]*redis.Z, error) {
	resList := make([]*redis.Z, 0, len(notificationEvents))
	for _, notificationEvent := range notificationEvents {
		toInsert, err := toInsertableItem(notificationEvent)
		if err != nil {
			return nil, err
		}
		resList = append(resList, toInsert)
	}
	return resList, nil
}

func toInsertableItem(notificationEvent *moira.NotificationEventHistoryItem) (*redis.Z, error) {
	notificationBytes, err := toNotificationBytes(notificationEvent)
	if err != nil {
		return nil, err
	}
	return &redis.Z{Score: float64(notificationEvent.TimeStamp), Member: notificationBytes}, nil
}

func storeNotificationHistoryBySingleKey(ctx context.Context, database moira.Database, toInsert []*redis.Z) error {
	switch db := database.(type) {
	case *moira_redis.DbConnector:
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

func prepareSplitItemsToInsert(eventsMap map[string][]*moira.NotificationEventHistoryItem) (map[string][]*redis.Z, error) {
	resMap := make(map[string][]*redis.Z, len(eventsMap))
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

func storeSplitNotifications(ctx context.Context, database moira.Database, toInsertMap map[string][]*redis.Z) error {
	switch db := database.(type) {
	case *moira_redis.DbConnector:
		client := db.Client()

		pipe := client.TxPipeline()

		for contactID, insertItems := range toInsertMap {
			key := contactNotificationKey + ":" + contactID
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
