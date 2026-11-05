package checker

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/holidayCalendar"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHolidayTriggerSuppression(t *testing.T) {
	calendar, err := holidayCalendar.New(holidayCalendar.Config{
		Enabled: true,
		File:    "../holidayCalendar/testdata/holidays.ics",
	})
	require.NoError(t, err)

	for _, test := range []struct {
		name             string
		timestamp        string
		offset           int64
		disabledMute     bool
		disabledCalendar bool
		missingConfig    bool
		weekdaysOnly     bool
		maintenance      bool
		want             bool
	}{
		{name: "holiday", timestamp: "2026-05-01T12:00:00Z", offset: -180, want: true},
		{name: "before local holiday", timestamp: "2026-04-30T20:59:59Z", offset: -180},
		{name: "local holiday starts", timestamp: "2026-04-30T21:00:00Z", offset: -180, want: true},
		{name: "same timestamp in UTC", timestamp: "2026-04-30T21:00:00Z"},
		{name: "local holiday ends", timestamp: "2026-05-01T21:00:00Z", offset: -180},
		{name: "offset west of UTC", timestamp: "2026-05-02T04:00:00Z", offset: 300, want: true},
		{name: "transferred day off", timestamp: "2026-01-09T12:00:00Z", offset: -180, want: true},
		{name: "ordinary weekend", timestamp: "2026-01-10T12:00:00Z", offset: -180},
		{name: "named holiday on weekend", timestamp: "2026-05-09T12:00:00Z", offset: -180, want: true},
		{name: "trigger flag disabled", timestamp: "2026-05-01T12:00:00Z", offset: -180, disabledMute: true},
		{name: "calendar disabled", timestamp: "2026-05-01T12:00:00Z", offset: -180, disabledCalendar: true},
		{name: "checker config absent", timestamp: "2026-05-01T12:00:00Z", offset: -180, missingConfig: true},
		{name: "schedule still suppresses weekends", timestamp: "2026-01-10T12:00:00Z", offset: -180, weekdaysOnly: true, want: true},
		{name: "maintenance still suppresses", timestamp: "2026-05-04T12:00:00Z", offset: -180, maintenance: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			schedule := moira.NewDefaultScheduleData()

			schedule.TimezoneOffset = test.offset

			if test.weekdaysOnly {
				schedule.Days[5].Enabled = false
				schedule.Days[6].Enabled = false
			}

			checker := &TriggerChecker{
				trigger: &moira.Trigger{MuteOnHolidays: !test.disabledMute, Schedule: schedule},
				config:  &Config{HolidayCalendar: calendar},
			}
			if test.disabledCalendar {
				checker.config.HolidayCalendar = nil
			}

			if test.missingConfig {
				checker.config = nil
			}

			timestamp := parseHolidayCheckTime(t, test.timestamp)

			maintenance := int64(0)

			if test.maintenance {
				maintenance = timestamp + 60
			}

			require.Equal(t, test.want, checker.isTriggerSuppressed(timestamp, maintenance))
		})
	}
}

func TestHolidayEventStateTransitions(t *testing.T) {
	calendar, err := holidayCalendar.New(holidayCalendar.Config{
		Enabled: true,
		File:    "../holidayCalendar/testdata/holidays.ics",
	})
	require.NoError(t, err)

	type checkStep struct {
		timestamp  string
		state      moira.State
		suppressed bool
		sendEvent  bool
	}

	for _, test := range []struct {
		name         string
		initialState moira.State
		weekdaysOnly bool
		steps        []checkStep
	}{
		{
			name:         "only the final persistent state is sent after holidays",
			initialState: moira.StateOK,
			steps: []checkStep{
				{"2026-04-30T21:00:00Z", moira.StateERROR, true, false},
				{"2026-05-01T12:00:00Z", moira.StateWARN, true, false},
				{"2026-05-01T21:00:00Z", moira.StateWARN, false, true},
				{"2026-05-01T21:01:00Z", moira.StateWARN, false, false},
			},
		},
		{
			name:         "a problem resolved during holidays is not replayed",
			initialState: moira.StateOK,
			steps: []checkStep{
				{"2026-04-30T21:00:00Z", moira.StateERROR, true, false},
				{"2026-05-01T12:00:00Z", moira.StateOK, false, false},
				{"2026-05-01T21:00:00Z", moira.StateOK, false, false},
			},
		},
		{
			name:         "recovery from a pre-holiday problem is sent afterward",
			initialState: moira.StateERROR,
			steps: []checkStep{
				{"2026-04-30T21:00:00Z", moira.StateOK, true, false},
				{"2026-05-01T21:00:00Z", moira.StateOK, false, true},
			},
		},
		{
			name:         "weekly schedule still applies after holidays",
			initialState: moira.StateOK,
			weekdaysOnly: true,
			steps: []checkStep{
				{"2026-05-01T10:00:00Z", moira.StateERROR, true, false},
				{"2026-05-02T10:00:00Z", moira.StateERROR, true, false},
				{"2026-05-04T06:00:00Z", moira.StateERROR, false, true},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, triggerEvent := range []bool{false, true} {
				name := "metric event"
				if triggerEvent {
					name = "trigger event"
				}

				t.Run(name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					db := mock_moira_alert.NewMockDatabase(ctrl)

					var events []moira.NotificationEvent

					db.EXPECT().PushNotificationEvent(gomock.Any(), true).
						DoAndReturn(func(event *moira.NotificationEvent, _ bool) error {
							events = append(events, *event)

							return nil
						}).AnyTimes()

					schedule := moira.NewDefaultScheduleData()
					if test.weekdaysOnly {
						schedule.StartOffset = 9 * 60
						schedule.EndOffset = 18 * 60
						schedule.Days[5].Enabled = false
						schedule.Days[6].Enabled = false
					}

					checker := &TriggerChecker{
						database:  db,
						triggerID: "holiday-trigger",
						trigger: &moira.Trigger{
							Name:           "Holiday trigger",
							MuteOnHolidays: true,
							Schedule:       schedule,
						},
						config:    &Config{HolidayCalendar: calendar},
						lastCheck: &moira.CheckData{},
					}
					beforeHoliday := parseHolidayCheckTime(t, "2026-04-30T20:59:00Z")
					previous := moira.MetricState{
						State:          test.initialState,
						Timestamp:      beforeHoliday,
						EventTimestamp: beforeHoliday,
					}
					wantEvents := 0

					for _, step := range test.steps {
						timestamp := parseHolidayCheckTime(t, step.timestamp)
						actual := compareHolidayState(t, checker, triggerEvent, previous, step.state, timestamp)
						require.Equal(t, step.state, actual.State, step.timestamp)
						require.Equal(t, step.suppressed, actual.Suppressed, step.timestamp)

						if step.suppressed {
							require.Equal(t, test.initialState, actual.SuppressedState, step.timestamp)
						}

						if step.sendEvent {
							wantEvents++
						}

						require.Len(t, events, wantEvents, step.timestamp)

						if step.sendEvent {
							event := events[len(events)-1]
							require.Equal(t, checker.triggerID, event.TriggerID)
							require.Equal(t, triggerEvent, event.IsTriggerEvent)
							require.Equal(t, test.initialState, event.OldState)
							require.Equal(t, step.state, event.State)
							require.Equal(t, timestamp, event.Timestamp)
						}

						previous = actual
					}
				})
			}
		})
	}
}

func compareHolidayState(
	t *testing.T,
	checker *TriggerChecker,
	triggerEvent bool,
	previous moira.MetricState,
	state moira.State,
	timestamp int64,
) moira.MetricState {
	t.Helper()

	if !triggerEvent {
		actual, err := checker.compareMetricStates("test.metric", moira.MetricState{State: state, Timestamp: timestamp}, previous)
		require.NoError(t, err)

		return actual
	}

	checker.lastCheck = &moira.CheckData{
		State:           previous.State,
		Timestamp:       previous.Timestamp,
		EventTimestamp:  previous.EventTimestamp,
		Suppressed:      previous.Suppressed,
		SuppressedState: previous.SuppressedState,
	}
	actual, err := checker.compareTriggerStates(moira.CheckData{State: state, Timestamp: timestamp})
	require.NoError(t, err)

	return moira.MetricState{
		State:           actual.State,
		Timestamp:       actual.Timestamp,
		EventTimestamp:  actual.EventTimestamp,
		Suppressed:      actual.Suppressed,
		SuppressedState: actual.SuppressedState,
	}
}

func parseHolidayCheckTime(t *testing.T, value string) int64 {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed.Unix()
}
