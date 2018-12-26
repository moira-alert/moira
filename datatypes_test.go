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

	Convey("No schedule", t, func() {
		var noSchedule *ScheduleData
		So(noSchedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Full schedule", t, func() {
		schedule := getDefaultSchedule()
		So(schedule.IsScheduleAllows(367980), ShouldBeTrue)
	})

	Convey("Exclude monday", t, func() {
		schedule := getDefaultSchedule()
		schedule.Days[0].Enabled = false
		So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		So(schedule.IsScheduleAllows(454380), ShouldBeTrue)
		So(schedule.IsScheduleAllows(367980+86400*2), ShouldBeTrue)
	})

	Convey("Exclude all days", t, func() {
		schedule := allDaysExcludedSchedule
		So(schedule.IsScheduleAllows(367980), ShouldBeFalse)
		So(schedule.IsScheduleAllows(454380), ShouldBeFalse)
		So(schedule.IsScheduleAllows(367980+86400*5), ShouldBeFalse)
	})

	Convey("Include only morning", t, func() {
		schedule := getDefaultSchedule()                           // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 60                                  // 01:00
		schedule.EndOffset = 540                                   // 09:00
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeTrue)  // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-241*60), ShouldBeFalse) // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeFalse) // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeFalse) // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Check border cases", t, func() {
		schedule := getDefaultSchedule()                    // TimeZone: Asia/Ekaterinburg (YEKT)
		So(schedule.IsScheduleAllows(68400), ShouldBeTrue)  // 02/01/1970 00:00:00 (YEKT)
		So(schedule.IsScheduleAllows(68401), ShouldBeTrue)  // 02/01/1970 00:00:01 (YEKT)
		So(schedule.IsScheduleAllows(68430), ShouldBeTrue)  // 02/01/1970 00:00:30 (YEKT)
		So(schedule.IsScheduleAllows(68459), ShouldBeTrue)  // 02/01/1970 00:00:59 (YEKT)
		So(schedule.IsScheduleAllows(154739), ShouldBeTrue) // 02/01/1970 23:58:59 (YEKT)
		So(schedule.IsScheduleAllows(154740), ShouldBeTrue) // 02/01/1970 23:59:00 (YEKT)
		So(schedule.IsScheduleAllows(154741), ShouldBeTrue) // 02/01/1970 23:59:01 (YEKT)
		So(schedule.IsScheduleAllows(154770), ShouldBeTrue) // 02/01/1970 23:59:30 (YEKT)
		So(schedule.IsScheduleAllows(154799), ShouldBeTrue) // 02/01/1970 23:59:59 (YEKT)
	})

	Convey("Exclude morning", t, func() {
		schedule := getDefaultSchedule()                           // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 420                                 // 07:00
		schedule.EndOffset = 1439                                  // 23:59
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeTrue)  // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeFalse) // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-242*60), ShouldBeFalse) // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeTrue)  // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeFalse) // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Exclude 10 minutes between 07:00 and 07:10", t, func() {
		schedule := getDefaultSchedule()                           // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 430                                 // 07:10
		schedule.EndOffset = 420                                   // 07:00
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeFalse) // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-242*60), ShouldBeTrue)  // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeTrue)  // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeTrue)  // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})

	Convey("Exclude business hours", t, func() {
		schedule := getDefaultSchedule()                           // TimeZone: Asia/Ekaterinburg (YEKT)
		schedule.StartOffset = 1200                                // 20:00
		schedule.EndOffset = 420                                   // 07:00
		So(schedule.IsScheduleAllows(86400+129*60), ShouldBeFalse) // 02/01/1970 2:09  - 02/01/1970 07:09 (YEKT)
		So(schedule.IsScheduleAllows(86400-239*60), ShouldBeTrue)  // 01/01/1970 20:01 - 02/01/1970 01:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-242*60), ShouldBeTrue)  // 01/01/1970 19:59 - 02/01/1970 00:59 (YEKT)
		So(schedule.IsScheduleAllows(86400+541*60), ShouldBeFalse) // 02/01/1970 9:01  - 02/01/1970 14:01 (YEKT)
		So(schedule.IsScheduleAllows(86400-255*60), ShouldBeTrue)  // 01/01/1970 19:45 - 02/01/1970 00:45 (YEKT)
	})
}

func TestEventsData_GetSubjectState(t *testing.T) {
	Convey("Get ERROR state", t, func() {
		message := "mes1"
		var value float64 = 1
		states := NotificationEvents{{State: "OK"}, {State: "ERROR", Message: &message, Value: &value}}
		So(states.GetSubjectState(), ShouldResemble, "ERROR")
		So(states[0].String(), ShouldResemble, "TriggerId: , Metric: , Value: 0, OldState: , State: OK, Message: '', Timestamp: 0")
		So(states[1].String(), ShouldResemble, "TriggerId: , Metric: , Value: 1, OldState: , State: ERROR, Message: 'mes1', Timestamp: 0")
	})
}

func TestTriggerData_GetTags(t *testing.T) {
	Convey("Test one tag", t, func() {
		triggerData := TriggerData{
			Tags: []string{"tag1"},
		}
		So(triggerData.GetTags(), ShouldResemble, "[tag1]")
	})
	Convey("Test many tags", t, func() {
		triggerData := TriggerData{
			Tags: []string{"tag1", "tag2", "tag...orNot"},
		}
		So(triggerData.GetTags(), ShouldResemble, "[tag1][tag2][tag...orNot]")
	})
	Convey("Test no tags", t, func() {
		triggerData := TriggerData{
			Tags: make([]string, 0),
		}
		So(triggerData.GetTags(), ShouldBeEmpty)
	})
}

func TestScheduledNotification_GetKey(t *testing.T) {
	Convey("Get key", t, func() {
		notification := ScheduledNotification{
			Contact:   ContactData{Type: "email", Value: "my@mail.com"},
			Event:     NotificationEvent{Value: nil, State: "NODATA", Metric: "my.metric"},
			Timestamp: 123456789,
		}
		So(notification.GetKey(), ShouldResemble, "email:my@mail.com::my.metric:NODATA:0:0.000000:0:false:123456789")
	})
}

func TestCheckData_GetOrCreateMetricState(t *testing.T) {
	Convey("Test no metric", t, func() {
		checkData := CheckData{
			Metrics: make(map[string]MetricState),
		}
		So(checkData.GetOrCreateMetricState("my.metric", 12343, false), ShouldResemble, MetricState{State: "NODATA", Timestamp: 12343})
	})
	Convey("Test no metric, notifyAboutNew = false", t, func() {
		checkData := CheckData{
			Metrics: make(map[string]MetricState),
		}
		So(checkData.GetOrCreateMetricState("my.metric", 12343, true), ShouldResemble, MetricState{State: "OK", Timestamp: time.Now().Unix(), EventTimestamp: time.Now().Unix()})
	})
	Convey("Test has metric", t, func() {
		metricState := MetricState{Timestamp: 11211}
		checkData := CheckData{
			Metrics: map[string]MetricState{
				"my.metric": metricState,
			},
		}
		So(checkData.GetOrCreateMetricState("my.metric", 12343, false), ShouldResemble, metricState)
	})
}

func TestMetricState_GetCheckPoint(t *testing.T) {
	Convey("Get check point", t, func() {
		metricState := MetricState{Timestamp: 800, EventTimestamp: 700}
		So(metricState.GetCheckPoint(120), ShouldEqual, 700)

		metricState = MetricState{Timestamp: 830, EventTimestamp: 700}
		So(metricState.GetCheckPoint(120), ShouldEqual, 710)

		metricState = MetricState{Timestamp: 699, EventTimestamp: 700}
		So(metricState.GetCheckPoint(1), ShouldEqual, 700)
	})
}

func TestMetricState_GetEventTimestamp(t *testing.T) {
	Convey("Get event timestamp", t, func() {
		metricState := MetricState{Timestamp: 800, EventTimestamp: 0}
		So(metricState.GetEventTimestamp(), ShouldEqual, 800)

		metricState = MetricState{Timestamp: 830, EventTimestamp: 700}
		So(metricState.GetEventTimestamp(), ShouldEqual, 700)
	})
}

func TestTrigger_IsSimple(t *testing.T) {
	Convey("Is Simple", t, func() {
		trigger := Trigger{
			Patterns: []string{"123"},
			Targets:  []string{"123"},
		}

		So(trigger.IsSimple(), ShouldBeTrue)
	})

	Convey("Not simple", t, func() {
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
			So(trigger.IsSimple(), ShouldBeFalse)
		}
	})
}

func TestCheckData_GetEventTimestamp(t *testing.T) {
	Convey("Get event timestamp", t, func() {
		checkData := CheckData{Timestamp: 800, EventTimestamp: 0}
		So(checkData.GetEventTimestamp(), ShouldEqual, 800)

		checkData = CheckData{Timestamp: 830, EventTimestamp: 700}
		So(checkData.GetEventTimestamp(), ShouldEqual, 700)
	})
}

func TestCheckData_UpdateScore(t *testing.T) {
	Convey("Update score", t, func() {
		checkData := CheckData{State: "NODATA"}
		So(checkData.UpdateScore(), ShouldEqual, 1000)
		So(checkData.Score, ShouldEqual, 1000)

		checkData = CheckData{
			State: "OK",
			Metrics: map[string]MetricState{
				"123": {State: "NODATA"},
				"321": {State: "OK"},
				"345": {State: "WARN"},
			},
		}
		So(checkData.UpdateScore(), ShouldEqual, 1001)
		So(checkData.Score, ShouldEqual, 1001)

		checkData = CheckData{
			State: "NODATA",
			Metrics: map[string]MetricState{
				"123": {State: "NODATA"},
				"321": {State: "OK"},
				"345": {State: "WARN"},
			},
		}
		So(checkData.UpdateScore(), ShouldEqual, 2001)
		So(checkData.Score, ShouldEqual, 2001)
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
		State    string
		OldState string
		Ignored  bool
	}
	assertIgnored := func(subscription SubscriptionData, eventCase testCase) {
		Convey(fmt.Sprintf("%s -> %s", eventCase.OldState, eventCase.State), func() {
			event := NotificationEvent{State: eventCase.State, OldState: eventCase.OldState}
			actual := subscription.MustIgnore(&event)
			So(actual, ShouldEqual, eventCase.Ignored)
		})
	}
	Convey("Has one type of transitions marked to be ignored", testing, func() {
		Convey("[TRUE] Send notifications when triggers degraded only", func() {
			subscription := SubscriptionData{
				Enabled:           true,
				IgnoreRecoverings: true,
				IgnoreWarnings:    false,
			}
			testCases := []testCase{
				{"WARN", "OK", false},
				{"ERROR", "OK", false},
				{"NODATA", "OK", false},
				{"ERROR", "WARN", false},
				{"NODATA", "WARN", false},
				{"NODATA", "ERROR", false},
				{"OK", "WARN", true},
				{"OK", "ERROR", true},
				{"OK", "NODATA", true},
				{"WARN", "ERROR", true},
				{"WARN", "NODATA", true},
				{"ERROR", "NODATA", true},
			}
			for _, testCase := range testCases {
				assertIgnored(subscription, testCase)
			}
		})
		Convey("[TRUE] Do not send WARN notifications", func() {
			subscription := SubscriptionData{
				Enabled:           true,
				IgnoreRecoverings: false,
				IgnoreWarnings:    true,
			}
			testCases := []testCase{
				{"ERROR", "OK", false},
				{"NODATA", "OK", false},
				{"ERROR", "WARN", false},
				{"NODATA", "WARN", false},
				{"NODATA", "ERROR", false},
				{"OK", "ERROR", false},
				{"OK", "NODATA", false},
				{"WARN", "ERROR", false},
				{"WARN", "NODATA", false},
				{"ERROR", "NODATA", false},
				{"OK", "WARN", true},
				{"WARN", "OK", true},
			}
			for _, testCase := range testCases {
				assertIgnored(subscription, testCase)
			}
		})
	})
	Convey("Has both types of transitions marked to be ignored", testing, func() {
		subscription := SubscriptionData{
			Enabled:           true,
			IgnoreRecoverings: true,
			IgnoreWarnings:    true,
		}
		testCases := []testCase{
			{"ERROR", "OK", false},
			{"NODATA", "OK", false},
			{"ERROR", "WARN", false},
			{"NODATA", "WARN", false},
			{"NODATA", "ERROR", false},
			{"OK", "WARN", true},
			{"WARN", "OK", true},
			{"OK", "ERROR", true},
			{"OK", "NODATA", true},
			{"WARN", "ERROR", true},
			{"WARN", "NODATA", true},
			{"ERROR", "NODATA", true},
		}
		for _, testCase := range testCases {
			assertIgnored(subscription, testCase)
		}
	})
	Convey("Has no types of transitions marked to be ignored", testing, func() {
		subscription := SubscriptionData{
			Enabled:           true,
			IgnoreRecoverings: false,
			IgnoreWarnings:    false,
		}
		testCases := []testCase{
			{"OK", "WARN", false},
			{"WARN", "OK", false},
			{"ERROR", "OK", false},
			{"NODATA", "OK", false},
			{"ERROR", "WARN", false},
			{"NODATA", "WARN", false},
			{"NODATA", "ERROR", false},
			{"OK", "ERROR", false},
			{"OK", "NODATA", false},
			{"WARN", "NODATA", false},
			{"ERROR", "NODATA", false},
		}
		for _, testCase := range testCases {
			assertIgnored(subscription, testCase)
		}
	})
}
