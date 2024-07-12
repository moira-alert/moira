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

func TestSplitNotificationHistory(t *testing.T) {
	conf := getDefault()
	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	if err != nil {
		t.Fatal(err)
	}
	db := moira_redis.NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	ctx := context.TODO()
	client := db.Client()

	eventsMap := eventsByKey(testNotificationHistoryEvents)

	Convey("Test split notification history", t, func() {
		Convey("with empty contactNotificationKey", func() {
			err = splitNotificationHistoryByContactId(ctx, logger, db, -1)
			So(err, ShouldBeNil)

			keys, err := client.Keys(ctx, contactNotificationKey+":*").Result()
			So(err, ShouldBeNil)
			So(keys, ShouldHaveLength, 0)
		})

		toInsert, err := prepareItemsForInsert(testNotificationHistoryEvents)
		So(err, ShouldBeNil)

		err = storeNotificationHistoryBySingleKey(ctx, db, toInsert)
		So(err, ShouldBeNil)

		Convey("with prepared history", func() {
			fetchCount := int64(len(toInsert) / 2)

			err = splitNotificationHistoryByContactId(ctx, logger, db, fetchCount)
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
	})
}

func prepareItemsForInsert(notificationEvents []*moira.NotificationEventHistoryItem) ([]*redis.Z, error) {
	toInsert := make([]*redis.Z, 0, len(notificationEvents))
	for _, event := range notificationEvents {
		notificationBytes, err := toNotificationBytes(event)
		if err != nil {
			return nil, err
		}
		toInsert = append(toInsert, &redis.Z{Score: float64(event.TimeStamp), Member: notificationBytes})
	}
	return toInsert, nil
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
