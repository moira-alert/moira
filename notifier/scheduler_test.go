package notifier

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var plottingData = moira.PlottingData{
	Enabled: false,
	Theme:   "dark",
}

func TestThrottling(t *testing.T) {
	trigger := moira.TriggerData{
		ID:         "triggerID-0000000000001",
		Name:       "test trigger",
		Targets:    []string{"test.target.5"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag"},
	}

	contact := moira.ContactData{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
	}

	subID := "SubscriptionID-000000000000001" //nolint

	event := moira.NotificationEvent{
		Metric:         "generate.event.1",
		State:          moira.StateOK,
		OldState:       moira.StateWARN,
		TriggerID:      trigger.ID,
		SubscriptionID: &subID,
	}

	subscription := moira.SubscriptionData{
		ID:                "SubscriptionID-000000000000001",
		Enabled:           true,
		Tags:              []string{"test-tag"},
		Contacts:          []string{"ContactID-000000000000001"},
		ThrottlingEnabled: true,
		Schedule:          schedule5,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")
	metrics2 := metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")

	now := time.Now()
	next := now.Add(10 * time.Minute)
	systemClock := mock_clock.NewMockClock(mockCtrl)
	scheduler := NewScheduler(dataBase, logger, metrics2, SchedulerConfig{ReschedulingDelay: time.Minute}, systemClock)

	params := moira.SchedulerParams{
		Event:        event,
		Trigger:      trigger,
		Contact:      contact,
		Plotting:     plottingData,
		ThrottledOld: false,
		SendFail:     0,
	}

	expected := moira.ScheduledNotification{
		Event:     event,
		Trigger:   trigger,
		Contact:   contact,
		Plotting:  plottingData,
		Throttled: false,
		Timestamp: now.Unix(),
		SendFail:  0,
		CreatedAt: now.Unix(),
	}

	Convey("Test sendFail more than 0, and no throttling, should send message in one minute", t, func() {
		params2 := params
		params2.ThrottledOld = false
		params2.SendFail = 1

		expected2 := expected
		expected2.SendFail = 1
		expected2.Timestamp = now.Add(time.Minute).Unix()
		systemClock.EXPECT().NowUTC().Return(now).Times(1)
		dataBase.EXPECT().GetTriggerThrottling(params2.Event.TriggerID).Return(now, now)
		dataBase.EXPECT().GetSubscription(*params2.Event.SubscriptionID).Return(subscription, nil)
		dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Unix(), 10), allTimeTo).Return(int64(0))
		dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Unix(), 10), allTimeTo).Return(int64(0))

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected2)
	})

	Convey("Test sendFail more that 0, and no throttling, but subscription doesn't exists, should send message in one minute", t, func() {
		params2 := params
		params2.ThrottledOld = false
		params2.SendFail = 1
		testErr := errors.New("subscription doesn't exist")

		expected2 := expected
		expected2.SendFail = 1
		expected2.Timestamp = now.Add(time.Minute).Unix()
		systemClock.EXPECT().NowUTC().Return(now).Times(1)
		dataBase.EXPECT().GetTriggerThrottling(params2.Event.TriggerID).Return(now, now)
		dataBase.EXPECT().GetSubscription(*params2.Event.SubscriptionID).Return(moira.SubscriptionData{}, testErr)

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected2)
	})

	Convey("Test sendFail more that 0, and no throttling, but the subscription schedule postpones the dispatch time, should send message in one minute", t, func() {
		params2 := params
		params2.ThrottledOld = false
		params2.SendFail = 1

		// 2015-09-02, 01:00:00 GMT+03:00
		testNow := time.Unix(1441144800, 0)
		testSubscription := subscription
		testSubscription.ThrottlingEnabled = false
		testSubscription.Schedule = schedule3

		expected2 := expected
		expected2.SendFail = 1
		// 2015-09-02, 02:00:00 GMT+03:00
		expected2.Timestamp = time.Unix(1441148400, 0).Unix()
		expected2.CreatedAt = testNow.Unix()
		systemClock.EXPECT().NowUTC().Return(testNow).Times(1)
		dataBase.EXPECT().GetTriggerThrottling(params2.Event.TriggerID).Return(testNow, testNow)
		dataBase.EXPECT().GetSubscription(*params2.Event.SubscriptionID).Return(testSubscription, nil)

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected2)
	})

	Convey("Test sendFail more than 0, and has throttling, should send message in one minute", t, func() {
		params2 := params
		params2.ThrottledOld = true
		params2.SendFail = 3

		expected2 := expected
		expected2.SendFail = 3
		expected2.Timestamp = now.Add(10 * time.Minute).Unix()
		expected2.Throttled = true

		systemClock.EXPECT().NowUTC().Return(now).Times(1)
		dataBase.EXPECT().GetTriggerThrottling(params2.Event.TriggerID).Return(next, now)
		dataBase.EXPECT().GetSubscription(*params2.Event.SubscriptionID).Return(subscription, nil)

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected2)
	})

	Convey("Test event state is TEST and no send fails, should return now notification time", t, func() {
		subID := "SubscriptionID-000000000000001"
		testEvent := moira.NotificationEvent{
			Metric:         "generate.event.1",
			State:          moira.StateTEST,
			OldState:       moira.StateWARN,
			TriggerID:      trigger.ID,
			SubscriptionID: &subID,
		}

		params2 := params
		params2.Event = testEvent

		expected3 := expected
		expected3.Event = testEvent

		systemClock.EXPECT().NowUTC().Return(now).Times(1)

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected3)
	})

	Convey("Test no throttling and no subscription, should return now notification time", t, func() {
		dataBase.EXPECT().GetTriggerThrottling(trigger.ID).Times(1).Return(time.Unix(0, 0), time.Unix(0, 0))
		dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Times(1).Return(moira.SubscriptionData{}, fmt.Errorf("Error while read subscription"))

		params2 := params

		systemClock.EXPECT().NowUTC().Return(now).Times(1)

		notification := scheduler.ScheduleNotification(params2, logger)
		So(notification, ShouldResemble, &expected)
	})
}

func TestSubscriptionSchedule(t *testing.T) {
	subID := "SubscriptionID-000000000000001"
	subscription := moira.SubscriptionData{
		ID:                "SubscriptionID-000000000000001",
		Enabled:           true,
		Tags:              []string{"test-tag"},
		Contacts:          []string{"ContactID-000000000000001"},
		ThrottlingEnabled: true,
	}

	event := moira.NotificationEvent{
		Metric:         "generate.event.1",
		State:          moira.StateOK,
		OldState:       moira.StateWARN,
		TriggerID:      "triggerID-0000000000001",
		SubscriptionID: &subID,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")
	notifierMetrics := metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")
	systemClock := mock_clock.NewMockClock(mockCtrl)
	scheduler := NewScheduler(dataBase, logger, notifierMetrics, SchedulerConfig{ReschedulingDelay: time.Minute}, systemClock)

	Convey("Throttling disabled", t, func() {
		now := time.Unix(1441187115, 0)
		subscription.ThrottlingEnabled = false

		Convey("When current time is allowed, should send notification now", func() {
			subscription.Schedule = schedule1

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
		})

		Convey("When allowed time is today, should send notification at the beginning of allowed interval", func() {
			subscription.Schedule = schedule2

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, time.Unix(1441191600, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("When allowed time is in a future day, should send notification at the beginning of allowed interval", func() {
			now = time.Unix(1441101600, 0)
			subscription.Schedule = schedule1

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, time.Unix(1441134000, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Trigger already alarm fatigue, but now throttling disabled, should send notification now", func() {
			subscription.Schedule = schedule1

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(1441187215, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeTrue)
		})
	})

	Convey("Throttling enabled", t, func() {
		now := time.Unix(1441134000, 0)
		subscription.ThrottlingEnabled = true

		Convey("Has trigger events count slightly less than low throttling level, should next timestamp now minutes, but throttling", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Add(-time.Hour*3).Unix(), 10), allTimeTo).Return(int64(13))
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Add(-time.Hour).Unix(), 10), allTimeTo).Return(int64(9))

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeTrue)
		})

		Convey("Has trigger events count event more than low throttling level, should next timestamp in 30 minutes", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Add(-time.Hour*3).Unix(), 10), allTimeTo).Return(int64(10))
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Add(-time.Hour).Unix(), 10), allTimeTo).Return(int64(10))
			dataBase.EXPECT().SetTriggerThrottling(event.TriggerID, now.Add(time.Hour/2)).Return(nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, time.Unix(1441135800, 0))
			So(throttled, ShouldBeTrue)
		})

		Convey("Has trigger event more than high throttling level, should next timestamp in 1 hour", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, strconv.FormatInt(now.Add(-time.Hour*3).Unix(), 10), allTimeTo).Return(int64(20))
			dataBase.EXPECT().SetTriggerThrottling(event.TriggerID, now.Add(time.Hour)).Return(nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, now.Add(time.Hour))
			So(throttled, ShouldBeTrue)
		})

		Convey("Trigger already alarm fatigue, should has old throttled value", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(1441148000, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			So(next, ShouldResemble, time.Unix(1441148000, 0))
			So(throttled, ShouldBeTrue)
		})
	})

	Convey("Test advanced schedule during current day (e.g. 02:00 - 00:00)", t, func() {
		// Schedule: 02:00 - 00:00 (GTM +3)
		Convey("Time is out of range, nextTime should resemble now", func() {
			// 2015-09-02, 14:00:00 GMT+03:00
			now := time.Unix(1441191600, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 14:00:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441191600, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Time is in range, nextTime should resemble start of new period", func() {
			// 2015-09-02, 01:00:00 GMT+03:00
			now := time.Unix(1441144800, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 02:00:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441148400, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Up border case, nextTime should resemble now", func() {
			// 2015-09-02, 02:00:00 GMT+03:00
			now := time.Unix(1441148400, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 02:00:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441148400, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Low border case, nextTime should resemble start of new period", func() {
			// 2015-09-02, 00:00:00 GMT+03:00
			now := time.Unix(1441141200, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 02:00:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441148400, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Low border case - 1 minute, nextTime should resemble now", func() {
			// 2015-09-01, 23:59:00 GMT+03:00
			now := time.Unix(1441141140, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-01, 23:59:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441141140, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Up border case - 1 minute, nextTime should resemble start of new period", func() {
			// 2015-09-02, 01:59:00 GMT+03:00
			now := time.Unix(1441148340, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule3

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 02:00:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441148400, 0))
			So(throttled, ShouldBeFalse)
		})
	})

	Convey("Test advanced schedule between different days (e.g. 23:30 - 18:00)", t, func() {
		// Schedule: 23:30 - 18:00 (GTM +3)
		Convey("Time is out of range within the current day, nextTime should resemble now", func() {
			// 2015-09-02, 23:45:00 GMT+03:00
			now := time.Unix(1441140300, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 23:45:00 GMT+03:00
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
		})

		Convey("Time is out of range on the next day, nextTime should resemble now", func() {
			// 2015-09-02, 00:35:00 GMT+03:00
			now := time.Unix(1441143300, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 00:35:00 GMT+03:00
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
		})

		Convey("Time is in range, nextTime should resemble start of new period", func() {
			// 2015-09-02, 20:00:00 GMT+03:00
			now := time.Unix(1441213200, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 23:30:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441225800, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Up border case, nextTime should resemble now", func() {
			// 2015-09-02, 23:30:00 GMT+03:00
			now := time.Unix(1441225800, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 23:30:00 GMT+03:00
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
		})

		Convey("Low border case, nextTime should resemble start of new period", func() {
			// 2015-09-02, 18:00:00 GMT+03:00
			now := time.Unix(1441206000, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 23:30:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441225800, 0))
			So(throttled, ShouldBeFalse)
		})

		Convey("Low border case - 1 minute, nextTime should resemble now", func() {
			// 2015-09-01, 17:59:00 GMT+03:00
			now := time.Unix(1441205940, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-01, 17:59:00 GMT+03:00
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
		})

		Convey("Up border case - 1 minute, nextTime should resemble start of new period", func() {
			// 2015-09-02, 23:29:00 GMT+03:00
			now := time.Unix(1441225740, 0)
			subscription.ThrottlingEnabled = false
			subscription.Schedule = schedule4

			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event, logger)
			// 2015-09-02, 23:30:00 GMT+03:00
			So(next, ShouldResemble, time.Unix(1441225800, 0))
			So(throttled, ShouldBeFalse)
		})
	})
}

var schedule1 = moira.ScheduleData{
	StartOffset:    0,   // 0:00 (GMT +5) after
	EndOffset:      900, // 15:00 (GMT +5)
	TimezoneOffset: -300,
	Days: []moira.ScheduleDataDay{
		{Enabled: false},
		{Enabled: false},
		{Enabled: true},
		{Enabled: false},
		{Enabled: false},
		{Enabled: false},
		{Enabled: false},
	},
}

var schedule2 = moira.ScheduleData{
	StartOffset:    660, // 16:00 (GMT +5) before
	EndOffset:      900, // 20:00 (GMT +5)
	TimezoneOffset: 0,
	Days: []moira.ScheduleDataDay{
		{Enabled: false},
		{Enabled: false},
		{Enabled: true},
		{Enabled: false},
		{Enabled: false},
		{Enabled: false},
		{Enabled: false},
	},
}

var schedule3 = moira.ScheduleData{
	StartOffset:    120,  // 02:00
	EndOffset:      0,    // 00:00
	TimezoneOffset: -180, // (GMT +3)
	Days: []moira.ScheduleDataDay{
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
	},
}

var schedule4 = moira.ScheduleData{
	StartOffset:    1410, // 23:30
	EndOffset:      1080, // 18:00
	TimezoneOffset: -180, // (GMT +3)
	Days: []moira.ScheduleDataDay{
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
	},
}

var schedule5 = moira.ScheduleData{
	StartOffset:    0,    // 00:00
	EndOffset:      1440, // 24:00
	TimezoneOffset: -180, // (GMT +3)
	Days: []moira.ScheduleDataDay{
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
		{Enabled: true},
	},
}
