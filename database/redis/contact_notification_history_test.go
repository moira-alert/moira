package redis

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const defaultTestMetric = "some_metric"

var inputScheduledNotification = moira.ScheduledNotification{
	Event: moira.NotificationEvent{
		IsTriggerEvent: true,
		Timestamp:      time.Now().Unix(),
		Metric:         defaultTestMetric,
		State:          moira.StateERROR,
		OldState:       moira.StateOK,
		TriggerID:      "1111-2222-33-4444-5555",
	},
	Trigger: moira.TriggerData{
		ID:   "1111-2222-33-4444-5555",
		Name: "Awesome Trigger",
		Desc: "No desc",
		Targets: []string{
			"some.metric.path",
		},
		WarnValue:  0.9,
		ErrorValue: 1.0,
		IsRemote:   false,
		Tags: []string{
			"TEST_TAG1",
			"TEST_TAG2",
		},
	},
	Contact: moira.ContactData{
		Type:  "slack",
		Value: "#auf_channel",
		ID:    "contact_id",
		User:  "user",
	},
	Plotting: moira.PlottingData{
		Enabled: false,
	},
	Throttled: false,
	SendFail:  1,
	Timestamp: time.Now().Unix(),
	CreatedAt: time.Now().Unix(),
}

var eventsShouldBeInDb = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: inputScheduledNotification.Timestamp,
		Metric:    inputScheduledNotification.Event.Metric,
		State:     inputScheduledNotification.Event.State,
		OldState:  inputScheduledNotification.Event.OldState,
		TriggerID: inputScheduledNotification.Trigger.ID,
		ContactID: inputScheduledNotification.Contact.ID,
	},
}

func TestGetNotificationsTotalByContactID(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	var contactID = "id"
	var from int64 = 0
	var to int64 = 1000

	Convey("GetNotificationsTotalByContactID", t, func() {

		Convey("When no notifications exist, total should be 0", func() {
			total, err := dataBase.GetNotificationsTotalByContactID(contactID, from, to)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 0)
		})

		Convey("After adding a notification, total should reflect it", func() {
			err := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
			So(err, ShouldBeNil)

			total, err := dataBase.GetNotificationsTotalByContactID(
				inputScheduledNotification.Contact.ID,
				inputScheduledNotification.Timestamp-5,
				inputScheduledNotification.Timestamp+5,
			)
			So(err, ShouldBeNil)
			So(total, ShouldEqual, 1)
		})
	})
}

func TestGetNotificationsHistoryByContactID(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)

	var defaultPage int64 = 0

	var defaultSize int64 = 100

	Convey("Notification history items manipulation", t, func() {
		dataBase.Flush()
		defer dataBase.Flush()

		Convey("While no data then notification items should be empty", func() {
			items, err := dataBase.GetNotificationsHistoryByContactID(
				"id",
				eventsShouldBeInDb[0].TimeStamp,
				eventsShouldBeInDb[0].TimeStamp,
				defaultPage,
				defaultSize)

			So(err, ShouldBeNil)
			So(items, ShouldHaveLength, 0)
		})

		Convey("Write event and check for success write", func() {
			errPushEvents := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
			So(errPushEvents, ShouldBeNil)

			Convey("Ensure that we can find event on +- 5 seconds interval", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp+5,
					defaultPage,
					defaultSize)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event exactly by its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp,
					defaultPage,
					defaultSize)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event if 'from' border equals its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp+5,
					defaultPage,
					defaultSize)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can find event if 'to' border equals its timestamp", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp,
					defaultPage,
					defaultSize)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that we can't find event time borders don't fit event timestamp", func() {
				veryOldFrom := int64(928930626) // 09.06.1999, 12:17:06
				veryOldTo := int64(992089026)   // 09.06.2001, 12:17:06

				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					veryOldFrom,
					veryOldTo,
					defaultPage,
					defaultSize)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldNotResemble, eventsShouldBeInDb)
			})

			Convey("Ensure that with negative page and positive size empty slice returned", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp,
					-1,
					1)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldHaveLength, 0)
			})

			Convey("Ensure that with positive page and negative size empty slice returned", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					eventsShouldBeInDb[0].TimeStamp,
					1,
					-1)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldHaveLength, 0)
			})

			otherScheduledNotification := inputScheduledNotification
			otherScheduledNotification.Timestamp += 1
			errPushEvents = dataBase.PushContactNotificationToHistory(&otherScheduledNotification)
			So(errPushEvents, ShouldBeNil)

			Convey("Ensure that with page=0 size=1 returns first event", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp+5,
					0,
					1)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, eventsShouldBeInDb)
			})

			otherEventShouldBeInDb := []*moira.NotificationEventHistoryItem{
				{
					TimeStamp: otherScheduledNotification.Timestamp,
					Metric:    otherScheduledNotification.Event.Metric,
					State:     otherScheduledNotification.Event.State,
					OldState:  otherScheduledNotification.Event.OldState,
					TriggerID: otherScheduledNotification.Trigger.ID,
					ContactID: otherScheduledNotification.Contact.ID,
				},
			}

			Convey("Ensure that with page=1 size=1 returns another event", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					eventsShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp-5,
					eventsShouldBeInDb[0].TimeStamp+5,
					1,
					1)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldResemble, otherEventShouldBeInDb)
			})

			Convey("Ensure that with page=0 size=-1 returns all events", func() {
				eventFromDb, err := dataBase.GetNotificationsHistoryByContactID(
					otherEventShouldBeInDb[0].ContactID,
					eventsShouldBeInDb[0].TimeStamp,
					otherEventShouldBeInDb[0].TimeStamp,
					0,
					-1)
				So(err, ShouldBeNil)
				So(eventFromDb, ShouldHaveLength, 2)
			})
		})
	})
}

func TestPushNotificationToHistory(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)

	Convey("Ensure that event would not have duplicates", t, func() {
		dataBase.Flush()
		defer dataBase.Flush()

		err1 := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
		So(err1, ShouldBeNil)

		err2 := dataBase.PushContactNotificationToHistory(&inputScheduledNotification)
		So(err2, ShouldBeNil)

		dbContent, err3 := dataBase.GetNotificationsHistoryByContactID(
			inputScheduledNotification.Contact.ID,
			inputScheduledNotification.Timestamp,
			inputScheduledNotification.Timestamp,
			0,
			100)

		So(err3, ShouldBeNil)
		So(dbContent, ShouldHaveLength, 1)
	})
}

var (
	testTTL = int64(48 * time.Hour)
	testNow = time.Now().Unix()
)

var outdatedEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testNow - testTTL - 1,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-1",
	},
	{
		TimeStamp: testNow - testTTL,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testNow - testTTL,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-1",
	},
	{
		TimeStamp: testNow - testTTL,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
}

var notOutdatedEvents = []*moira.NotificationEventHistoryItem{
	{
		TimeStamp: testNow,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-2",
	},
	{
		TimeStamp: testNow,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-1",
	},
	{
		TimeStamp: testNow,
		Metric:    defaultTestMetric,
		State:     moira.StateTEST,
		OldState:  "",
		TriggerID: "",
		ContactID: "contact-id-3",
	},
}

func TestCleanUpOutdatedNotificationHistory(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()

	defer dataBase.Flush()

	Convey("Test clean up notification history", t, func() {
		Convey("with empty database", func() {
			err := dataBase.CleanUpOutdatedNotificationHistory(testTTL)
			So(err, ShouldBeNil)
		})

		Convey("with prepared events", func() {
			storeErr := storeNotificationHistoryItems(dataBase, append(outdatedEvents, notOutdatedEvents...))
			So(storeErr, ShouldBeNil)

			err := dataBase.CleanUpOutdatedNotificationHistory(testTTL)
			So(err, ShouldBeNil)

			client := dataBase.Client()

			contactIDs, errKeys := client.Keys(dataBase.context, contactNotificationKeyWithID("*")).Result()
			So(errKeys, ShouldBeNil)
			So(contactIDs, ShouldHaveLength, len(notOutdatedEvents))

			eventsMap := toEventsMap(notOutdatedEvents)

			for _, contactID := range contactIDs {
				Convey(fmt.Sprintf("for contact with id: %s", contactID), func() {
					events, errGet := dataBase.GetNotificationsHistoryByContactID(contactID, testNow-testTTL, testNow, 0, -1)
					So(errGet, ShouldBeNil)
					So(events, ShouldHaveLength, len(eventsMap[contactID]))

					for i := range events {
						So(events[i], ShouldResemble, eventsMap[contactID][i])
					}
				})
			}
		})
	})
}

func storeNotificationHistoryItems(connector *DbConnector, notificationEvents []*moira.NotificationEventHistoryItem) error {
	client := connector.Client()

	pipe := client.TxPipeline()

	for _, notification := range notificationEvents {
		notificationBytes, err := GetNotificationBytes(notification)
		if err != nil {
			return err
		}

		pipe.ZAdd(
			connector.context,
			contactNotificationKeyWithID(notification.ContactID),
			&redis.Z{
				Score:  float64(notification.TimeStamp),
				Member: notificationBytes,
			})
	}

	_, err := pipe.Exec(connector.context)

	return err
}

func toEventsMap(events []*moira.NotificationEventHistoryItem) map[string][]*moira.NotificationEventHistoryItem {
	m := make(map[string][]*moira.NotificationEventHistoryItem, len(events))
	for _, event := range events {
		m[event.ContactID] = append(m[event.ContactID], event)
	}

	return m
}

func TestDbConnector_CountEventsInNotificationHistory(t *testing.T) {
	Convey("Test counting events in notification history", t, func() {
		logger, _ := logging.GetLogger("dataBase")
		dataBase := NewTestDatabase(logger)
		dataBase.Flush()

		defer dataBase.Flush()

		baseTimeStamp := time.Now().Unix()

		contactIDs := []string{
			"first-id",
			"second-id",
			"third-id",
		}

		timestampRanges := []int64{
			baseTimeStamp - 4,
			baseTimeStamp - 1,
			baseTimeStamp,
			baseTimeStamp + 1,
		}

		notificationHistoryItems := make([]*moira.NotificationEventHistoryItem, 0, len(contactIDs)*len(timestampRanges))

		for _, contactID := range contactIDs {
			for _, timestamp := range timestampRanges {
				notificationHistoryItems = append(notificationHistoryItems,
					&moira.NotificationEventHistoryItem{
						ContactID: contactID,
						TimeStamp: timestamp,
					})
			}
		}

		notificationHistoryItems[len(timestampRanges)].TimeStamp = baseTimeStamp - 2

		err := storeNotificationHistoryItems(dataBase, notificationHistoryItems)
		So(err, ShouldBeNil)

		Convey("with one contactID", func() {
			Convey("events from -inf to +inf", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(contactIDs[:1], allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 4},
				})
			})

			Convey("events from fixed timestamp to +inf", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(
					contactIDs[:1],
					strconv.FormatInt(baseTimeStamp-2, 10),
					allTimeTo)
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 3},
				})
			})

			Convey("events from fixed timestamp to fixed timestamp", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(
					contactIDs[:1],
					strconv.FormatInt(baseTimeStamp-2, 10),
					strconv.FormatInt(baseTimeStamp, 10))
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 2},
				})
			})

			Convey("events from -inf to fixed timestamp", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(
					contactIDs[:1],
					allTimeFrom,
					strconv.FormatInt(baseTimeStamp-2, 10))
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 1},
				})
			})
		})

		Convey("with several contactID", func() {
			Convey("all time", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(contactIDs[:2], allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 4},
					{ID: contactIDs[1], Count: 4},
				})
			})

			Convey("events from fixed timestamp to +inf", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(contactIDs[:2],
					strconv.FormatInt(baseTimeStamp-3, 10),
					allTimeTo)
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 3},
					{ID: contactIDs[1], Count: 4},
				})
			})

			Convey("events from fixed timestamp to fixed timestamp", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(contactIDs[:2],
					strconv.FormatInt(baseTimeStamp-3, 10),
					strconv.FormatInt(baseTimeStamp, 10))
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 2},
					{ID: contactIDs[1], Count: 3},
				})
			})

			Convey("events from -inf to fixed timestamp", func() {
				eventCounts, err := dataBase.CountEventsInNotificationHistory(contactIDs[:2],
					allTimeFrom,
					strconv.FormatInt(baseTimeStamp, 10))
				So(err, ShouldBeNil)
				So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
					{ID: contactIDs[0], Count: 3},
					{ID: contactIDs[1], Count: 3},
				})
			})
		})

		Convey("with no existing contact id", func() {
			eventCounts, err := dataBase.CountEventsInNotificationHistory(
				append(contactIDs[:2], "forth-id"),
				allTimeFrom,
				allTimeTo)
			So(err, ShouldBeNil)
			So(eventCounts, ShouldResemble, []*moira.ContactIDWithNotificationCount{
				{ID: contactIDs[0], Count: 4},
				{ID: contactIDs[1], Count: 4},
				{ID: "forth-id", Count: 0},
			})
		})
	})
}
