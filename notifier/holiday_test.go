package notifier

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/holidayCalendar"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHolidayNotificationDelivery(t *testing.T) {
	calendar, err := holidayCalendar.New(holidayCalendar.Config{
		Enabled: true,
		File:    "../holidayCalendar/testdata/holidays.ics",
	})
	require.NoError(t, err)

	logger, err := logging.GetLogger("HolidaySchedulerTest")
	require.NoError(t, err)

	for _, test := range []struct {
		name             string
		now              string
		want             string
		offset           int64
		days             string
		disabledMute     bool
		disabledCalendar bool
		testEvent        bool
		sendFail         int
		throttleDelay    time.Duration
	}{
		{name: "holiday delays delivery", now: "2026-05-01T10:00:00Z", want: "2026-05-01T21:00:00Z", offset: -180},
		{name: "holiday ends in subscription timezone", now: "2026-05-01T10:00:00Z", want: "2026-05-02T00:00:00Z"},
		{name: "local holiday already started", now: "2026-04-30T21:00:00Z", want: "2026-05-01T21:00:00Z", offset: -180},
		{name: "same instant is not a holiday in UTC", now: "2026-04-30T21:00:00Z", want: "2026-04-30T21:00:00Z"},
		{name: "ordinary weekend remains allowed", now: "2026-01-10T12:00:00Z", want: "2026-01-10T12:00:00Z", offset: -180},
		{name: "holiday on weekend is muted", now: "2026-05-09T12:00:00Z", want: "2026-05-09T21:00:00Z", offset: -180},
		{name: "adjacent holidays are skipped together", now: "2025-12-31T12:00:00Z", want: "2026-01-09T21:00:00Z", offset: -180},
		{name: "flag disabled", now: "2026-05-01T10:00:00Z", want: "2026-05-01T10:00:00Z", offset: -180, disabledMute: true},
		{name: "calendar disabled", now: "2026-05-01T10:00:00Z", want: "2026-05-01T10:00:00Z", offset: -180, disabledCalendar: true},
		{name: "weekly schedule applies after holiday", now: "2026-05-01T10:00:00Z", want: "2026-05-04T06:00:00Z", offset: -180, days: "weekdays"},
		{name: "schedule lands on another holiday", now: "2026-05-08T16:00:00Z", want: "2026-05-12T06:00:00Z", offset: -180, days: "weekdays"},
		{name: "only weekends allowed", now: "2026-05-09T10:00:00Z", want: "2026-05-10T06:00:00Z", offset: -180, days: "weekends"},
		{name: "transferred day then ordinary Saturday", now: "2026-01-09T10:00:00Z", want: "2026-01-10T06:00:00Z", offset: -180, days: "weekends"},
		{name: "next allowed weekday is a holiday", now: "2026-04-30T16:00:00Z", want: "2026-05-08T06:00:00Z", offset: -180, days: "friday"},
		{name: "schedule still applies with flag disabled", now: "2026-05-02T10:00:00Z", want: "2026-05-04T06:00:00Z", offset: -180, days: "weekdays", disabledMute: true},
		{name: "schedule still applies with calendar disabled", now: "2026-05-02T10:00:00Z", want: "2026-05-04T06:00:00Z", offset: -180, days: "weekdays", disabledCalendar: true},
		{name: "empty schedule still uses its offset", now: "2026-04-30T21:00:00Z", want: "2026-05-01T21:00:00Z", offset: -180, days: "empty"},
		{name: "retry crosses holiday boundary", now: "2026-04-30T20:59:30Z", want: "2026-05-01T21:00:00Z", offset: -180, sendFail: 1},
		{name: "retry on ordinary day keeps delay", now: "2026-05-04T10:00:00Z", want: "2026-05-04T10:01:00Z", offset: -180, sendFail: 1},
		{name: "initial TEST event is immediate", now: "2026-05-01T10:00:00Z", want: "2026-05-01T10:00:00Z", offset: -180, testEvent: true},
		{name: "failed TEST event follows holiday calendar", now: "2026-05-01T10:00:00Z", want: "2026-05-01T21:00:00Z", offset: -180, testEvent: true, sendFail: 1},
		{name: "throttling crosses holiday boundary", now: "2026-04-30T20:30:00Z", want: "2026-05-01T21:00:00Z", offset: -180, throttleDelay: time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := parseHolidayDeliveryTime(t, test.now)
			schedule := moira.NewDefaultScheduleData()
			schedule.TimezoneOffset = test.offset

			if test.days == "empty" {
				schedule.Days = nil
			} else if test.days != "" {
				schedule.StartOffset = 9 * 60
				schedule.EndOffset = 18 * 60

				for day := range schedule.Days {
					switch test.days {
					case "weekdays":
						schedule.Days[day].Enabled = day < 5
					case "weekends":
						schedule.Days[day].Enabled = day >= 5
					case "friday":
						schedule.Days[day].Enabled = day == 4
					}
				}
			}

			subscription := moira.SubscriptionData{
				ID:                "holiday-subscription",
				Enabled:           true,
				MuteOnHolidays:    !test.disabledMute,
				Schedule:          *schedule,
				ThrottlingEnabled: test.throttleDelay > 0,
			}
			params := moira.SchedulerParams{
				Event: moira.NotificationEvent{
					TriggerID:      "holiday-trigger",
					SubscriptionID: &subscription.ID,
					Metric:         "test.metric",
					OldState:       moira.StateOK,
					State:          moira.StateERROR,
					Timestamp:      now.Add(-time.Minute).Unix(),
				},
				Trigger:  moira.TriggerData{ID: "holiday-trigger", Name: "Holiday trigger"},
				Contact:  moira.ContactData{ID: "holiday-contact", Type: "email", Value: "test@example.com"},
				Plotting: moira.PlottingData{Enabled: true, Theme: "dark"},
				SendFail: test.sendFail,
			}

			if test.testEvent {
				params.Event.State = moira.StateTEST
			}

			ctrl := gomock.NewController(t)
			db := mock_moira_alert.NewMockDatabase(ctrl)
			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().NowUTC().Return(now)

			if !test.testEvent || test.sendFail > 0 {
				db.EXPECT().GetTriggerThrottling(params.Event.TriggerID).Return(now.Add(test.throttleDelay), now)
				db.EXPECT().GetSubscription(subscription.ID).Return(subscription, nil)
			}

			config := SchedulerConfig{HolidayCalendar: calendar, ReschedulingDelay: time.Minute}
			if test.disabledCalendar {
				config.HolidayCalendar = nil
			}

			scheduler := NewScheduler(db, logger, nil, config, clock)
			notification := scheduler.ScheduleNotification(params, logger)

			require.Equal(t, &moira.ScheduledNotification{
				Event:     params.Event,
				Trigger:   params.Trigger,
				Contact:   params.Contact,
				Plotting:  params.Plotting,
				SendFail:  params.SendFail,
				Throttled: test.throttleDelay > 0,
				CreatedAt: now.Unix(),
				Timestamp: parseHolidayDeliveryTime(t, test.want).Unix(),
			}, notification)
		})
	}
}

func TestHolidaySubscriptionRejectsInvalidSchedule(t *testing.T) {
	calendar, err := holidayCalendar.New(holidayCalendar.Config{
		Enabled: true,
		File:    "../holidayCalendar/testdata/holidays.ics",
	})
	require.NoError(t, err)

	for _, test := range []struct {
		name string
		want string
	}{
		{name: "incomplete week", want: "invalid scheduled settings: 2 days defined"},
		{name: "no enabled days", want: "can not find allowed schedule day"},
	} {
		t.Run(test.name, func(t *testing.T) {
			schedule := moira.NewDefaultScheduleData()
			if test.name == "incomplete week" {
				schedule.Days = schedule.Days[:2]
			} else {
				for day := range schedule.Days {
					schedule.Days[day].Enabled = false
				}
			}

			scheduler := &StandardScheduler{config: SchedulerConfig{HolidayCalendar: calendar}}
			_, err := scheduler.applySubscriptionSchedule(&moira.SubscriptionData{
				MuteOnHolidays: true,
				Schedule:       *schedule,
			}, parseHolidayDeliveryTime(t, "2026-05-01T10:00:00Z"))
			require.EqualError(t, err, test.want)
		})
	}
}

func parseHolidayDeliveryTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}
