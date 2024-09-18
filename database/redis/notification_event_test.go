package redis

import (
	"strconv"
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

const (
	triggerID  = "81588c33-eab3-4ad4-aa03-82a9560adad9"
	triggerID1 = "7854DE02-0E4B-4430-A570-B0C0162755E4"
	triggerID2 = "26D3C4E4-507E-4930-9B1E-FD5AD369445C"
	triggerID3 = "F0F4A5B9-637C-4933-AA0D-88B9798A2630" //nolint
)

var (
	allTimeFrom = "-inf"
	allTimeTo   = "+inf"
	now         = time.Now().Unix()
	value       = float64(0)
)

// nolint
func TestNotificationEvents(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	Convey("Notification events manipulation", t, func() {
		Convey("Test push-get-get count-fetch", func() {
			Convey("Should no events", func() {
				actual, err := dataBase.GetNotificationEvents(triggerID, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))

				total := dataBase.GetNotificationEventCount(triggerID, 0)
				So(total, ShouldEqual, 0)

				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{})
			})

			Convey("Should has one events after push", func() {
				err := dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateNODATA,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID,
					Metric:    "my.metric",
					Value:     &value,
				}, true)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(triggerID, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateNODATA,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID,
						Metric:    "my.metric",
						Values:    map[string]float64{"t1": 0},
					},
				})

				total := dataBase.GetNotificationEventCount(triggerID, 0)
				So(total, ShouldEqual, 1)

				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateNODATA,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID,
					Metric:    "my.metric",
					Values:    map[string]float64{"t1": 0},
				})
			})

			Convey("Should has event by triggerID after fetch", func() {
				actual, err := dataBase.GetNotificationEvents(triggerID, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateNODATA,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID,
						Metric:    "my.metric",
						Values:    map[string]float64{"t1": 0},
					},
				})

				total := dataBase.GetNotificationEventCount(triggerID, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Should no events to fetch after fetch", func() {
				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test push-fetch multiple event by differ triggerIDs", func() {
			Convey("Push events and get it by triggerIDs", func() {
				err := dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateEXCEPTION,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID1,
					Metric:    "my.metric",
				}, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateOK,
					OldState:  moira.StateWARN,
					TriggerID: triggerID2,
					Metric:    "my.metric1",
					Value:     &value,
				}, true)
				So(err, ShouldBeNil)

				actual, err := dataBase.GetNotificationEvents(triggerID1, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateEXCEPTION,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID1,
						Metric:    "my.metric",
						Values:    map[string]float64{},
					},
				})

				total := dataBase.GetNotificationEventCount(triggerID1, 0)
				So(total, ShouldEqual, 1)

				actual, err = dataBase.GetNotificationEvents(triggerID2, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateOK,
						OldState:  moira.StateWARN,
						TriggerID: triggerID2,
						Metric:    "my.metric1",
						Values:    map[string]float64{"t1": 0},
					},
				})

				total = dataBase.GetNotificationEventCount(triggerID2, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Fetch one of them and check for existing again", func() {
				actual1, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual1, ShouldResemble, moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateEXCEPTION,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID1,
					Metric:    "my.metric",
					Values:    map[string]float64{},
				})

				actual, err := dataBase.GetNotificationEvents(triggerID1, 0, 1, allTimeFrom, allTimeTo)
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateEXCEPTION,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID1,
						Metric:    "my.metric",
						Values:    map[string]float64{},
					},
				})

				total := dataBase.GetNotificationEventCount(triggerID1, 0)
				So(total, ShouldEqual, 1)
			})

			Convey("Fetch second then fetch and and check for ErrNil", func() {
				actual, err := dataBase.FetchNotificationEvent()
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateOK,
					OldState:  moira.StateWARN,
					TriggerID: triggerID2,
					Metric:    "my.metric1",
					Values:    map[string]float64{"t1": 0},
				})

				actual, err = dataBase.FetchNotificationEvent()
				So(err, ShouldBeError)
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.NotificationEvent{})
			})
		})

		Convey("Test get by ranges", func() {
			err := dataBase.PushNotificationEvent(&moira.NotificationEvent{
				Timestamp: now,
				State:     moira.StateNODATA,
				OldState:  moira.StateNODATA,
				TriggerID: triggerID3,
				Metric:    "my.metric",
			}, true)
			So(err, ShouldBeNil)

			actual, err := dataBase.GetNotificationEvents(triggerID3, 0, 1, allTimeFrom, allTimeTo)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, []*moira.NotificationEvent{
				{
					Timestamp: now,
					State:     moira.StateNODATA,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID3,
					Metric:    "my.metric",
					Values:    map[string]float64{},
				},
			})

			total := dataBase.GetNotificationEventCount(triggerID3, 0)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(triggerID3, now-1)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(triggerID3, now)
			So(total, ShouldEqual, 1)

			total = dataBase.GetNotificationEventCount(triggerID3, now+1)
			So(total, ShouldEqual, 0)

			actual, err = dataBase.GetNotificationEvents(triggerID3, 1, 1, allTimeFrom, allTimeTo)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, make([]*moira.NotificationEvent, 0))
		})

		Convey("Test `from` and `to` params", func() {
			err := dataBase.PushNotificationEvent(&moira.NotificationEvent{
				Timestamp: now,
				State:     moira.StateNODATA,
				OldState:  moira.StateNODATA,
				TriggerID: triggerID3,
				Metric:    "my.metric",
			}, true)
			So(err, ShouldBeNil)

			Convey("returns event on exact time", func() {
				actual, err := dataBase.GetNotificationEvents(triggerID3, 0, 1, strconv.FormatInt(now, 10), strconv.FormatInt(now, 10))
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateNODATA,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID3,
						Metric:    "my.metric",
						Values:    map[string]float64{},
					},
				})
			})

			Convey("not return event out of time range", func() {
				actual, err := dataBase.GetNotificationEvents(triggerID3, 0, 1, strconv.FormatInt(now-2, 10), strconv.FormatInt(now-1, 10))
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{})
			})

			Convey("returns event in time range", func() {
				actual, err := dataBase.GetNotificationEvents(triggerID3, 0, 1, strconv.FormatInt(now-1, 10), strconv.FormatInt(now+1, 10))
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, []*moira.NotificationEvent{
					{
						Timestamp: now,
						State:     moira.StateNODATA,
						OldState:  moira.StateNODATA,
						TriggerID: triggerID3,
						Metric:    "my.metric",
						Values:    map[string]float64{},
					},
				})
			})
		})

		Convey("Test removing notification events", func() {
			Convey("Should remove all notifications", func() {
				err := dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateNODATA,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID,
					Metric:    "my.metric",
					Value:     &value, //nolint
				}, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateEXCEPTION,
					OldState:  moira.StateNODATA,
					TriggerID: triggerID1,
					Metric:    "my.metric",
					Value:     &value,
				}, true)
				So(err, ShouldBeNil)

				err = dataBase.PushNotificationEvent(&moira.NotificationEvent{
					Timestamp: now,
					State:     moira.StateOK,
					OldState:  moira.StateWARN,
					TriggerID: triggerID2,
					Metric:    "my.metric1",
					Value:     &value,
				}, true)
				So(err, ShouldBeNil)

				err = dataBase.RemoveAllNotificationEvents()
				So(err, ShouldBeNil)

				actual, err := dataBase.FetchNotificationEvent()
				So(err, ShouldResemble, database.ErrNil)
				So(actual, ShouldResemble, moira.NotificationEvent{})
			})
		})
	})
}

func TestNotificationEventErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()

	// TODO(litleleprikon): check why notification is event created here again
	newNotificationEvent := moira.NotificationEvent{
		Timestamp: time.Now().Unix(),
		State:     moira.StateNODATA,
		OldState:  moira.StateNODATA,
		TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
		Metric:    "my.metric",
	}

	Convey("Should throw error when no connection", t, func() {
		actual1, err := dataBase.GetNotificationEvents("123", 0, 1, allTimeFrom, allTimeTo)
		So(actual1, ShouldBeNil)
		So(err, ShouldNotBeNil)

		err = dataBase.PushNotificationEvent(&newNotificationEvent, true)
		So(err, ShouldNotBeNil)

		total := dataBase.GetNotificationEventCount("123", 0)
		So(total, ShouldEqual, 0)

		actual2, err := dataBase.FetchNotificationEvent()
		So(actual2, ShouldResemble, moira.NotificationEvent{})
		So(err, ShouldNotBeNil)
	})
}
