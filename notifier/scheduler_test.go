package notifier

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestThrottling(t *testing.T) {
	var trigger = moira.TriggerData{
		ID:         "triggerID-0000000000001",
		Name:       "test trigger",
		Targets:    []string{"test.target.5"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag"},
	}

	var contact = moira.ContactData{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
	}

	subID := "SubscriptionID-000000000000001"

	var event = moira.NotificationEvent{
		Metric:         "generate.event.1",
		State:          "OK",
		OldState:       "WARN",
		TriggerID:      trigger.ID,
		SubscriptionID: &subID,
	}

	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")
	scheduler := NewScheduler(dataBase, logger)

	now := time.Now()

	expected := moira.ScheduledNotification{
		Event:     event,
		Trigger:   trigger,
		Contact:   contact,
		Throttled: false,
		Timestamp: now.Unix(),
		SendFail:  0,
	}

	Convey("Test sendFail more than 0, and no throttling, should send message in one minute", t, func() {
		expected2 := expected
		expected2.SendFail = 1
		expected2.Timestamp = now.Add(time.Minute).Unix()

		notification := scheduler.ScheduleNotification(now, event, trigger, contact, false, 1)
		So(notification, ShouldResemble, &expected2)
		mockCtrl.Finish()
	})

	Convey("Test sendFail more than 0, and has throttling, should send message in one minute", t, func() {
		expected2 := expected
		expected2.SendFail = 3
		expected2.Timestamp = now.Add(time.Minute).Unix()
		expected2.Throttled = true

		notification := scheduler.ScheduleNotification(now, event, trigger, contact, true, 3)
		So(notification, ShouldResemble, &expected2)
		mockCtrl.Finish()
	})

	Convey("Test event state is TEST and no send fails, should return now notification time", t, func() {
		subID := "SubscriptionID-000000000000001"
		testEvent := moira.NotificationEvent{
			Metric:         "generate.event.1",
			State:          "TEST",
			OldState:       "WARN",
			TriggerID:      trigger.ID,
			SubscriptionID: &subID,
		}

		expected3 := expected
		expected3.Event = testEvent

		notification := scheduler.ScheduleNotification(now, testEvent, trigger, contact, false, 0)
		So(notification, ShouldResemble, &expected3)
		mockCtrl.Finish()
	})

	Convey("Test no throttling and no subscription, should return now notification time", t, func() {
		dataBase.EXPECT().GetTriggerThrottling(trigger.ID).Times(1).Return(time.Unix(0, 0), time.Unix(0, 0))
		dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Times(1).Return(moira.SubscriptionData{}, fmt.Errorf("Error while read subscription"))

		notification := scheduler.ScheduleNotification(now, event, trigger, contact, false, 0)
		So(notification, ShouldResemble, &expected)
		mockCtrl.Finish()
	})
}

func TestSubscriptionSchedule(t *testing.T) {
	subID := "SubscriptionID-000000000000001"
	var subscription = moira.SubscriptionData{
		ID:                "SubscriptionID-000000000000001",
		Enabled:           true,
		Tags:              []string{"test-tag"},
		Contacts:          []string{"ContactID-000000000000001"},
		ThrottlingEnabled: true,
	}

	var event = moira.NotificationEvent{
		Metric:         "generate.event.1",
		State:          "OK",
		OldState:       "WARN",
		TriggerID:      "triggerID-0000000000001",
		SubscriptionID: &subID,
	}

	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")
	scheduler := NewScheduler(dataBase, logger)

	Convey("Throttling disabled", t, func() {
		now := time.Unix(1441187115, 0)
		subscription.ThrottlingEnabled = false
		Convey("When current time is allowed, should send notification now", func() {
			subscription.Schedule = schedule1
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeFalse)
			mockCtrl.Finish()
		})

		Convey("When allowed time is today, should send notification at the beginning of allowed interval", func() {
			subscription.Schedule = schedule2
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, time.Unix(1441191600, 0))
			So(throttled, ShouldBeFalse)
			mockCtrl.Finish()
		})

		Convey("When allowed time is in a future day, should send notification at the beginning of allowed interval", func() {
			now = time.Unix(1441101600, 0)
			subscription.Schedule = schedule1
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, time.Unix(1441134000, 0))
			So(throttled, ShouldBeFalse)
			mockCtrl.Finish()
		})

		Convey("Trigger already alarm fatigue, but now throttling disabled, should send notification now", func() {
			subscription.Schedule = schedule1
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(1441187215, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeTrue)
			mockCtrl.Finish()
		})
	})

	Convey("Throttling enabled", t, func() {
		now := time.Unix(1441134000, 0)
		subscription.ThrottlingEnabled = true

		Convey("Has trigger events count slightly less than low throttling level, should next timestamp now minutes, but throttling", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, now.Add(-time.Hour*3).Unix()).Return(int64(13))
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, now.Add(-time.Hour).Unix()).Return(int64(9))

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, now)
			So(throttled, ShouldBeTrue)
			mockCtrl.Finish()
		})

		Convey("Has trigger events count event more than low throttling level, should next timestamp in 30 minutes", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, now.Add(-time.Hour*3).Unix()).Return(int64(10))
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, now.Add(-time.Hour).Unix()).Return(int64(10))
			dataBase.EXPECT().SetTriggerThrottling(event.TriggerID, now.Add(time.Hour/2)).Return(nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, time.Unix(1441135800, 0))
			So(throttled, ShouldBeTrue)
			mockCtrl.Finish()
		})

		Convey("Has trigger event more than high throttling level, should next timestamp in 1 hour", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(0, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)
			dataBase.EXPECT().GetNotificationEventCount(event.TriggerID, now.Add(-time.Hour*3).Unix()).Return(int64(20))
			dataBase.EXPECT().SetTriggerThrottling(event.TriggerID, now.Add(time.Hour)).Return(nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, now.Add(time.Hour))
			So(throttled, ShouldBeTrue)
			mockCtrl.Finish()
		})

		Convey("Trigger already alarm fatigue, should has old throttled value", func() {
			dataBase.EXPECT().GetTriggerThrottling(event.TriggerID).Return(time.Unix(1441148000, 0), time.Unix(0, 0))
			dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Return(subscription, nil)

			next, throttled := scheduler.calculateNextDelivery(now, &event)
			So(next, ShouldResemble, time.Unix(1441148000, 0))
			So(throttled, ShouldBeTrue)
			mockCtrl.Finish()
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
