package moira

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsScheduleAllows(t *testing.T) {
	allDaysExcludedSchedule := ScheduleData{
		TimezoneOffset: -300, // TimeZone: Asia/Ekaterinburg
		StartOffset:    0,    // 00:00
		EndOffset:      1439, // 23:59
		Days: []ScheduleDataDay{
			{
				Name:    "Mon",
				Enabled: false,
			},
			{
				Name:    "Tue",
				Enabled: false,
			},
			{
				Name:    "Wed",
				Enabled: false,
			},
			{
				Name:    "Thu",
				Enabled: false,
			},
			{
				Name:    "Fri",
				Enabled: false,
			},
			{
				Name:    "Sat",
				Enabled: false,
			},
			{
				Name:    "Sun",
				Enabled: false,
			},
		},
	}

	// Date Format: dd/mm/yyyy 24h:MM
	// 367980 - 05/01/1970 18:13 (UTC) Mon - 23:13 (YEKT)
	// 454380 - 06/01/1970 18:13 (UTC) Tue - 23:13 (YEKT)

	Convey("No schedule", t, func(c C) {
		var noSchedule *ScheduleData
		c.So(noSchedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Full schedule", t, func(c C) {
		schedule := getDefaultSchedule()
		c.So(schedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Exclude monday", t, func(c C) {
		schedule := getDefaultSchedule()
		schedule.Days[0].Enabled = false
		c.So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		c.So(schedule.IsScheduleAllows(454380), ShouldBeTrue)
		c.So(schedule.IsScheduleAllows(367980+86400*2), ShouldBeTrue)
	})

	Convey("Exclude all days", t, func(c C) {
		schedule := allDaysExcludedSchedule
		c.So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		c.So(schedule.IsScheduleAllows(454380), ShouldBeFalse)
		c.So(schedule.IsScheduleAllows(367980+86400*5), ShouldBeFalse)
	})

	Convey("Include only morning", t, func(c C) {
		schedule := getDefaultSchedule()                             // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 60                                    // 01:00
		schedule.EndOffset = 540                                     // 09:00
		c.So(schedule.IsScheduleAllows(86400+129*60), ShouldBeTrue)  // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-241*60), ShouldBeFalse) // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		c.So(schedule.IsScheduleAllows(86400+541*60), ShouldBeFalse) // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-255*60), ShouldBeFalse) // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Check border cases", t, func(c C) {
		schedule := getDefaultSchedule()                      // TimeZone: Asia/Ekaterinburg (YEKT)
		c.So(schedule.IsScheduleAllows(68400), ShouldBeTrue)  // 02/01/1970 00:00:00 (YEKT)
		c.So(schedule.IsScheduleAllows(68401), ShouldBeTrue)  // 02/01/1970 00:00:01 (YEKT)
		c.So(schedule.IsScheduleAllows(68430), ShouldBeTrue)  // 02/01/1970 00:00:30 (YEKT)
		c.So(schedule.IsScheduleAllows(68459), ShouldBeTrue)  // 02/01/1970 00:00:59 (YEKT)
		c.So(schedule.IsScheduleAllows(154739), ShouldBeTrue) // 02/01/1970 23:58:59 (YEKT)
		c.So(schedule.IsScheduleAllows(154740), ShouldBeTrue) // 02/01/1970 23:59:00 (YEKT)
		c.So(schedule.IsScheduleAllows(154741), ShouldBeTrue) // 02/01/1970 23:59:01 (YEKT)
		c.So(schedule.IsScheduleAllows(154770), ShouldBeTrue) // 02/01/1970 23:59:30 (YEKT)
		c.So(schedule.IsScheduleAllows(154799), ShouldBeTrue) // 02/01/1970 23:59:59 (YEKT)
	})

	Convey("Exclude morning", t, func(c C) {
		schedule := getDefaultSchedule()                             // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 420                                   // 07:00
		schedule.EndOffset = 1439                                    // 23:59
		c.So(schedule.IsScheduleAllows(86400+129*60), ShouldBeTrue)  // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-239*60), ShouldBeFalse) // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-242*60), ShouldBeFalse) // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		c.So(schedule.IsScheduleAllows(86400+541*60), ShouldBeTrue)  // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-255*60), ShouldBeFalse) // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Exclude 10 minutes between 07:00 and 07:10", t, func(c C) {
		schedule := getDefaultSchedule()                             // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 430                                   // 07:10
		schedule.EndOffset = 420                                     // 07:00
		c.So(schedule.IsScheduleAllows(86400+129*60), ShouldBeFalse) // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-242*60), ShouldBeTrue)  // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		c.So(schedule.IsScheduleAllows(86400+541*60), ShouldBeTrue)  // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-255*60), ShouldBeTrue)  // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Exclude business hours", t, func(c C) {
		schedule := getDefaultSchedule()                             // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 1200                                  // 20:00
		schedule.EndOffset = 420                                     // 07:00
		c.So(schedule.IsScheduleAllows(86400+129*60), ShouldBeFalse) // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-242*60), ShouldBeTrue)  // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		c.So(schedule.IsScheduleAllows(86400+541*60), ShouldBeFalse) // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		c.So(schedule.IsScheduleAllows(86400-255*60), ShouldBeTrue)  // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})
}

func TestNotificationEvent_GetSubjectState(t *testing.T) {
	Convey("Get ERROR state", t, func(c C) {
		message := "mes1"
		var value float64 = 1
		states := NotificationEvents{{State: StateOK}, {State: StateERROR, Message: &message, Value: &value}}
		c.So(states.GetSubjectState(), ShouldResemble, StateERROR)
		c.So(states[0].String(), ShouldResemble, "TriggerId: , Metric: , Value: 0, OldState: , State: OK, Message: '', Timestamp: 0")
		c.So(states[1].String(), ShouldResemble, "TriggerId: , Metric: , Value: 1, OldState: , State: ERROR, Message: 'mes1', Timestamp: 0")
	})
}

func TestNotificationEvent_FormatTimestamp(t *testing.T) {
	Convey("Test FormatTimestamp", t, func(c C) {
		event := NotificationEvent{Timestamp: 150000000}
		location, _ := time.LoadLocation("UTC")
		location1, _ := time.LoadLocation("Europe/Moscow")
		location2, _ := time.LoadLocation("Asia/Yekaterinburg")
		c.So(event.FormatTimestamp(location), ShouldResemble, "02:40")
		c.So(event.FormatTimestamp(location1), ShouldResemble, "05:40")
		c.So(event.FormatTimestamp(location2), ShouldResemble, "07:40")
	})
}

func TestNotificationEvent_GetValue(t *testing.T) {
	event := NotificationEvent{}
	value1 := float64(2.32)
	value2 := float64(2.3222222)
	value3 := float64(2)
	value4 := float64(2.000001)
	value5 := float64(2.33333333)
	Convey("Test GetMetricValue", t, func(c C) {
		c.So(event.GetMetricValue(), ShouldResemble, "0")
		event.Value = &value1
		c.So(event.GetMetricValue(), ShouldResemble, "2.32")
		event.Value = &value2
		c.So(event.GetMetricValue(), ShouldResemble, "2.3222222")
		event.Value = &value3
		c.So(event.GetMetricValue(), ShouldResemble, "2")
		event.Value = &value4
		c.So(event.GetMetricValue(), ShouldResemble, "2.000001")
		event.Value = &value5
		c.So(event.GetMetricValue(), ShouldResemble, "2.33333333")
	})
}

func TestTriggerData_GetTags(t *testing.T) {
	Convey("Test one tag", t, func(c C) {
		triggerData := TriggerData{
			Tags: []string{"tag1"},
		}
		c.So(triggerData.GetTags(), ShouldResemble, "[tag1]")
	})
	Convey("Test many tags", t, func(c C) {
		triggerData := TriggerData{
			Tags: []string{"tag1", "tag2", "tag...orNot"},
		}
		c.So(triggerData.GetTags(), ShouldResemble, "[tag1][tag2][tag...orNot]")
	})
	Convey("Test no tags", t, func(c C) {
		triggerData := TriggerData{
			Tags: make([]string, 0),
		}
		c.So(triggerData.GetTags(), ShouldBeEmpty)
	})
}

func TestScheduledNotification_GetKey(t *testing.T) {
	Convey("Get key", t, func(c C) {
		notification := ScheduledNotification{
			Contact:   ContactData{Type: "email", Value: "my@mail.com"},
			Event:     NotificationEvent{Value: nil, State: StateNODATA, Metric: "my.metric"},
			Timestamp: 123456789,
		}
		c.So(notification.GetKey(), ShouldResemble, "email:my@mail.com::my.metric:NODATA:0:0.000000:0:false:123456789")
	})
}

func TestCheckData_GetOrCreateMetricState(t *testing.T) {
	Convey("Test no metric", t, func(c C) {
		checkData := CheckData{
			Metrics: make(map[string]MetricState),
		}
		c.So(checkData.GetOrCreateMetricState("my.metric", 12343, false), ShouldResemble, MetricState{State: StateNODATA, Timestamp: 12343})
	})
	Convey("Test no metric, notifyAboutNew = false", t, func(c C) {
		checkData := CheckData{
			Metrics: make(map[string]MetricState),
		}
		c.So(checkData.GetOrCreateMetricState("my.metric", 12343, true), ShouldResemble, MetricState{State: StateOK, Timestamp: time.Now().Unix(), EventTimestamp: time.Now().Unix()})
	})
	Convey("Test has metric", t, func(c C) {
		metricState := MetricState{Timestamp: 11211}
		checkData := CheckData{
			Metrics: map[string]MetricState{
				"my.metric": metricState,
			},
		}
		c.So(checkData.GetOrCreateMetricState("my.metric", 12343, false), ShouldResemble, metricState)
	})
}

func TestMetricState_GetCheckPoint(t *testing.T) {
	Convey("Get check point", t, func(c C) {
		metricState := MetricState{Timestamp: 800, EventTimestamp: 700}
		c.So(metricState.GetCheckPoint(120), ShouldEqual, 700)

		metricState = MetricState{Timestamp: 830, EventTimestamp: 700}
		c.So(metricState.GetCheckPoint(120), ShouldEqual, 710)

		metricState = MetricState{Timestamp: 699, EventTimestamp: 700}
		c.So(metricState.GetCheckPoint(1), ShouldEqual, 700)
	})
}

func TestMetricState_GetEventTimestamp(t *testing.T) {
	Convey("Get event timestamp", t, func(c C) {
		metricState := MetricState{Timestamp: 800, EventTimestamp: 0}
		c.So(metricState.GetEventTimestamp(), ShouldEqual, 800)

		metricState = MetricState{Timestamp: 830, EventTimestamp: 700}
		c.So(metricState.GetEventTimestamp(), ShouldEqual, 700)
	})
}

func TestTrigger_IsSimple(t *testing.T) {
	Convey("Is Simple", t, func(c C) {
		trigger := Trigger{
			Patterns: []string{"123"},
			Targets:  []string{"123"},
		}

		c.So(trigger.IsSimple(), ShouldBeTrue)
	})

	Convey("Not simple", t, func(c C) {
		triggers := []Trigger{
			{Patterns: []string{"123", "1233"}},
			{Patterns: []string{"123", "1233"}, Targets: []string{"123", "1233"}},
			{Targets: []string{"123", "1233"}},
			{Patterns: []string{"123"}, Targets: []string{"123", "1233"}},
			{Patterns: []string{"123?"}, Targets: []string{"123"}},
			{Patterns: []string{"12*3"}, Targets: []string{"123"}},
			{Patterns: []string{"1{23"}, Targets: []string{"123"}},
			{Patterns: []string{"[123"}, Targets: []string{"123"}},
			{Patterns: []string{"[12*3"}, Targets: []string{"123"}},
		}

		for _, trigger := range triggers {
			c.So(trigger.IsSimple(), ShouldBeFalse)
		}
	})
}

func TestCheckData_GetEventTimestamp(t *testing.T) {
	Convey("Get event timestamp", t, func(c C) {
		checkData := CheckData{Timestamp: 800, EventTimestamp: 0}
		c.So(checkData.GetEventTimestamp(), ShouldEqual, 800)

		checkData = CheckData{Timestamp: 830, EventTimestamp: 700}
		c.So(checkData.GetEventTimestamp(), ShouldEqual, 700)
	})
}

func TestCheckData_UpdateScore(t *testing.T) {
	Convey("Update score", t, func(c C) {
		checkData := CheckData{State: StateNODATA}
		c.So(checkData.UpdateScore(), ShouldEqual, 1000)
		c.So(checkData.Score, ShouldEqual, 1000)

		checkData = CheckData{
			State: StateOK,
			Metrics: map[string]MetricState{
				"123": {State: StateNODATA},
				"321": {State: StateOK},
				"345": {State: StateWARN},
			},
		}
		c.So(checkData.UpdateScore(), ShouldEqual, 1001)
		c.So(checkData.Score, ShouldEqual, 1001)

		checkData = CheckData{
			State: StateNODATA,
			Metrics: map[string]MetricState{
				"123": {State: StateNODATA},
				"321": {State: StateOK},
				"345": {State: StateWARN},
			},
		}
		c.So(checkData.UpdateScore(), ShouldEqual, 2001)
		c.So(checkData.Score, ShouldEqual, 2001)
	})
}

func getDefaultSchedule() ScheduleData {
	return ScheduleData{
		TimezoneOffset: -300, // TimeZone: Asia/Ekaterinburg
		StartOffset:    0,    // 00:00
		EndOffset:      1439, // 23:59
		Days: []ScheduleDataDay{
			{
				Name:    "Mon",
				Enabled: true,
			},
			{
				Name:    "Tue",
				Enabled: true,
			},
			{
				Name:    "Wed",
				Enabled: true,
			},
			{
				Name:    "Thu",
				Enabled: true,
			},
			{
				Name:    "Fri",
				Enabled: true,
			},
			{
				Name:    "Sat",
				Enabled: true,
			},
			{
				Name:    "Sun",
				Enabled: true,
			},
		},
	}
}

func TestSubscriptionData_MustIgnore(testing *testing.T) {
	type testCase struct {
		State    State
		OldState State
		Ignored  bool
	}
	assertIgnored := func(subscription SubscriptionData, eventCase testCase) {
		Convey(fmt.Sprintf("%s -> %s", eventCase.OldState, eventCase.State), testing, func(c C) {
			event := NotificationEvent{State: eventCase.State, OldState: eventCase.OldState}
			actual := subscription.MustIgnore(&event)
			c.So(actual, ShouldEqual, eventCase.Ignored)
		})
	}
	Convey("Has one type of transitions marked to be ignored", testing, func(c C) {
		Convey("[TRUE] Send notifications when triggers degraded only", testing, func(c C) {
			subscription := SubscriptionData{
				Enabled:           true,
				IgnoreRecoverings: true,
				IgnoreWarnings:    false,
			}
			testCases := []testCase{
				{StateWARN, StateOK, false},
				{StateERROR, StateOK, false},
				{StateNODATA, StateOK, false},
				{StateERROR, StateWARN, false},
				{StateNODATA, StateWARN, false},
				{StateNODATA, StateERROR, false},
				{StateOK, StateWARN, true},
				{StateOK, StateERROR, true},
				{StateOK, StateNODATA, true},
				{StateWARN, StateERROR, true},
				{StateWARN, StateNODATA, true},
				{StateERROR, StateNODATA, true},
			}
			for _, testCase := range testCases {
				assertIgnored(subscription, testCase)
			}
		})
		Convey("[TRUE] Do not send WARN notifications", testing, func(c C) {
			subscription := SubscriptionData{
				Enabled:           true,
				IgnoreRecoverings: false,
				IgnoreWarnings:    true,
			}
			testCases := []testCase{
				{StateERROR, StateOK, false},
				{StateNODATA, StateOK, false},
				{StateERROR, StateWARN, false},
				{StateNODATA, StateWARN, false},
				{StateNODATA, StateERROR, false},
				{StateOK, StateERROR, false},
				{StateOK, StateNODATA, false},
				{StateWARN, StateERROR, false},
				{StateWARN, StateNODATA, false},
				{StateERROR, StateNODATA, false},
				{StateOK, StateWARN, true},
				{StateWARN, StateOK, true},
			}
			for _, testCase := range testCases {
				assertIgnored(subscription, testCase)
			}
		})
	})
	Convey("Has both types of transitions marked to be ignored", testing, func(c C) {
		subscription := SubscriptionData{
			Enabled:           true,
			IgnoreRecoverings: true,
			IgnoreWarnings:    true,
		}
		testCases := []testCase{
			{StateERROR, StateOK, false},
			{StateNODATA, StateOK, false},
			{StateERROR, StateWARN, false},
			{StateNODATA, StateWARN, false},
			{StateNODATA, StateERROR, false},
			{StateOK, StateWARN, true},
			{StateWARN, StateOK, true},
			{StateOK, StateERROR, true},
			{StateOK, StateNODATA, true},
			{StateWARN, StateERROR, true},
			{StateWARN, StateNODATA, true},
			{StateERROR, StateNODATA, true},
		}
		for _, testCase := range testCases {
			assertIgnored(subscription, testCase)
		}
	})
	Convey("Has no types of transitions marked to be ignored", testing, func(c C) {
		subscription := SubscriptionData{
			Enabled:           true,
			IgnoreRecoverings: false,
			IgnoreWarnings:    false,
		}
		testCases := []testCase{
			{StateOK, StateWARN, false},
			{StateWARN, StateOK, false},
			{StateERROR, StateOK, false},
			{StateNODATA, StateOK, false},
			{StateERROR, StateWARN, false},
			{StateNODATA, StateWARN, false},
			{StateNODATA, StateERROR, false},
			{StateOK, StateERROR, false},
			{StateOK, StateNODATA, false},
			{StateWARN, StateNODATA, false},
			{StateERROR, StateNODATA, false},
		}
		for _, testCase := range testCases {
			assertIgnored(subscription, testCase)
		}
	})
}
func TestBuildTriggerURL(t *testing.T) {
	Convey("Sender has no moira uri", t, func(c C) {
		url := TriggerData{ID: "SomeID"}.GetTriggerURI("")
		c.So(url, ShouldResemble, "/trigger/SomeID")
	})

	Convey("Sender uri", t, func(c C) {
		url := TriggerData{ID: "SomeID"}.GetTriggerURI("https://my-moira.com")
		c.So(url, ShouldResemble, "https://my-moira.com/trigger/SomeID")
	})

	Convey("Empty trigger", t, func(c C) {
		url := TriggerData{}.GetTriggerURI("https://my-moira.com")
		c.So(url, ShouldBeEmpty)
	})
}

func TestSetMaintenanceUserAndTime(t *testing.T) {
	startMaintenanceUser := "testStartMtUser"
	startMaintenanceUserOld := "testStartMtUserOld"
	stopMaintenanceUser := "testStopMtUser"
	stopMaintenanceUserOld := "testStopMtUserOld"
	callTime := int64(3000)

	Convey("Test MaintenanceInfo", t, func(c C) {
		testStartMaintenance(
			t,
			"Not MaintenanceInfo, user anonymous.",
			MaintenanceInfo{},
			"anonymous",
			MaintenanceInfo{},
		)

		testStopMaintenance(
			t,
			"Not MaintenanceInfo, user anonymous.",
			MaintenanceInfo{},
			"anonymous",
			MaintenanceInfo{},
		)

		testStartMaintenance(
			t,
			"Not MaintenanceInfo, user real.",
			MaintenanceInfo{},
			startMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUser, &callTime, nil, nil},
		)

		testStopMaintenance(
			t,
			"Not MaintenanceInfo, user real",
			MaintenanceInfo{},
			stopMaintenanceUser,
			MaintenanceInfo{nil, nil, &stopMaintenanceUser, &callTime},
		)

		testStartMaintenance(
			t,
			"Set Start in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{},
			"anonymous",
			MaintenanceInfo{},
		)

		testStopMaintenance(
			t,
			"Set Start in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{},
			"anonymous",
			MaintenanceInfo{},
		)

		testStartMaintenance(
			t,
			"Set Start in MaintenanceInfo, user real.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, nil, nil},
			startMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUser, &callTime, nil, nil},
		)

		testStopMaintenance(
			t,
			"Set Start in MaintenanceInfo, user real.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, nil, nil},
			stopMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUser, &callTime},
		)

		testStartMaintenance(
			t,
			"Set Stop in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{nil, nil, &stopMaintenanceUserOld, &callTime},
			"anonymous",
			MaintenanceInfo{},
		)

		testStopMaintenance(
			t,
			"Set Stop in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{nil, nil, &stopMaintenanceUserOld, &callTime},
			"anonymous",
			MaintenanceInfo{},
		)

		testStartMaintenance(
			t,
			"Set Stop in MaintenanceInfo, user real.",
			MaintenanceInfo{nil, nil, &stopMaintenanceUserOld, &callTime},
			startMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUser, &callTime, nil, nil},
		)

		testStopMaintenance(
			t,
			"Set Stop in MaintenanceInfo, user real.",
			MaintenanceInfo{nil, nil, &stopMaintenanceUserOld, &callTime},
			stopMaintenanceUser,
			MaintenanceInfo{nil, nil, &stopMaintenanceUser, &callTime},
		)

		testStartMaintenance(
			t,
			"Set Start and Stop in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUserOld, &callTime},
			"anonymous",
			MaintenanceInfo{},
		)

		testStopMaintenance(
			t,
			"Set Start and Stop in MaintenanceInfo, user anonymous.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUserOld, &callTime},
			"anonymous",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, nil, nil},
		)

		testStartMaintenance(
			t,
			"Set Start and Stop in MaintenanceInfo, user real.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUserOld, &callTime},
			startMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUser, &callTime, nil, nil},
		)

		testStopMaintenance(
			t,
			"Set Start and Stop in MaintenanceInfo, user real.",
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUserOld, &callTime},
			stopMaintenanceUser,
			MaintenanceInfo{&startMaintenanceUserOld, &callTime, &stopMaintenanceUser, &callTime},
		)
	})
}

func testStartMaintenance(t *testing.T, message string, actualInfo MaintenanceInfo, user string, expectedInfo MaintenanceInfo) {
	conveyMessage := fmt.Sprintf("%v Start maintenance.", message)
	testMaintenance(t, conveyMessage, actualInfo, 3100, user, expectedInfo)
}

func testStopMaintenance(t *testing.T, message string, actualInfo MaintenanceInfo, user string, expectedInfo MaintenanceInfo) {
	conveyMessage := fmt.Sprintf("%v Stop maintenance.", message)
	testMaintenance(t, conveyMessage, actualInfo, 0, user, expectedInfo)
}

func testMaintenance(t *testing.T, conveyMessage string, actualInfo MaintenanceInfo, maintenance int64, user string, expectedInfo MaintenanceInfo) {

	Convey(conveyMessage, t, func(c C) {
		var lastCheckTest = CheckData{
			Maintenance: 1000,
		}
		lastCheckTest.MaintenanceInfo = actualInfo

		SetMaintenanceUserAndTime(&lastCheckTest, maintenance, user, 3000)

		c.So(lastCheckTest.MaintenanceInfo, ShouldResemble, expectedInfo)
		c.So(lastCheckTest.Maintenance, ShouldEqual, maintenance)

	})
}
