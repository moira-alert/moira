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

func TestNotificationEvent_GetSubjectState(t *testing.T) {
	Convey("Get ERROR state", t, func() {
		message := "mes1"
		var value float64 = 1
		states := NotificationEvents{{State: StateOK}, {State: StateERROR, Message: &message, Value: &value}}
		So(states.GetSubjectState(), ShouldResemble, StateERROR)
		So(states[0].String(), ShouldResemble, "TriggerId: , Metric: , Value: 0, OldState: , State: OK, Message: '', Timestamp: 0")
		So(states[1].String(), ShouldResemble, "TriggerId: , Metric: , Value: 1, OldState: , State: ERROR, Message: 'mes1', Timestamp: 0")
	})
}

func TestNotificationEvent_FormatTimestamp(t *testing.T) {
	Convey("Test FormatTimestamp", t, func() {
		event := NotificationEvent{Timestamp: 150000000}
		location, _ := time.LoadLocation("UTC")
		location1, _ := time.LoadLocation("Europe/Moscow")
		location2, _ := time.LoadLocation("Asia/Yekaterinburg")
		So(event.FormatTimestamp(location), ShouldResemble, "02:40")
		So(event.FormatTimestamp(location1), ShouldResemble, "05:40")
		So(event.FormatTimestamp(location2), ShouldResemble, "07:40")
	})
}

func TestNotificationEvent_GetValue(t *testing.T) {
	event := NotificationEvent{}
	value1 := float64(2.32)
	value2 := float64(2.3222222)
	value3 := float64(2)
	value4 := float64(2.000001)
	value5 := float64(2.33333333)
	Convey("Test GetMetricValue", t, func() {
		So(event.GetMetricValue(), ShouldResemble, "0")
		event.Value = &value1
		So(event.GetMetricValue(), ShouldResemble, "2.32")
		event.Value = &value2
		So(event.GetMetricValue(), ShouldResemble, "2.3222222")
		event.Value = &value3
		So(event.GetMetricValue(), ShouldResemble, "2")
		event.Value = &value4
		So(event.GetMetricValue(), ShouldResemble, "2.000001")
		event.Value = &value5
		So(event.GetMetricValue(), ShouldResemble, "2.33333333")
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
			Event:     NotificationEvent{Value: nil, State: StateNODATA, Metric: "my.metric"},
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
		So(checkData.GetOrCreateMetricState("my.metric", 12343, false), ShouldResemble, MetricState{State: StateNODATA, Timestamp: 12343})
	})
	Convey("Test no metric, notifyAboutNew = false", t, func() {
		checkData := CheckData{
			Metrics: make(map[string]MetricState),
		}
		So(checkData.GetOrCreateMetricState("my.metric", 12343, true), ShouldResemble, MetricState{State: StateOK, Timestamp: time.Now().Unix(), EventTimestamp: time.Now().Unix()})
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
		checkData := CheckData{State: StateNODATA}
		So(checkData.UpdateScore(), ShouldEqual, 1000)
		So(checkData.Score, ShouldEqual, 1000)

		checkData = CheckData{
			State: StateOK,
			Metrics: map[string]MetricState{
				"123": {State: StateNODATA},
				"321": {State: StateOK},
				"345": {State: StateWARN},
			},
		}
		So(checkData.UpdateScore(), ShouldEqual, 1001)
		So(checkData.Score, ShouldEqual, 1001)

		checkData = CheckData{
			State: StateNODATA,
			Metrics: map[string]MetricState{
				"123": {State: StateNODATA},
				"321": {State: StateOK},
				"345": {State: StateWARN},
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
		State    State
		OldState State
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
		Convey("[TRUE] Do not send WARN notifications", func() {
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
	Convey("Has both types of transitions marked to be ignored", testing, func() {
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
	Convey("Has no types of transitions marked to be ignored", testing, func() {
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
	Convey("Sender has no moira uri", t, func() {
		url := TriggerData{ID: "SomeID"}.GetTriggerURI("")
		So(url, ShouldResemble, "/trigger/SomeID")
	})

	Convey("Sender uri", t, func() {
		url := TriggerData{ID: "SomeID"}.GetTriggerURI("https://my-moira.com")
		So(url, ShouldResemble, "https://my-moira.com/trigger/SomeID")
	})

	Convey("Empty trigger", t, func() {
		url := TriggerData{}.GetTriggerURI("https://my-moira.com")
		So(url, ShouldBeEmpty)
	})
}

func TestSetMaintenanceUserAndTime (t *testing.T) {
	startMaintenanceUser := "testStartMtUser"
	startMaintenanceTime := int64(1550304140)
	stopMaintenanceUser := "testStopMtUser"
	stopMaintenanceTime := int64(1553068940)
	triggerMaintenanceTS := int64(1552723340)

	Convey("Test trigger", t, func() {
		Convey("User anonymous", func() {
			actual := lastCheckTest
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, "anonymous", startMaintenanceTime)
			actual.Maintenance = triggerMaintenanceTS
			actual.MaintenanceInfo.Set(nil, nil, nil, nil)
			So(actual, ShouldResemble, lastCheckTest)
		})
		Convey("User '' ", func() {
			actual := lastCheckTest
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, "", startMaintenanceTime)
			actual.MaintenanceInfo.Set(nil, nil, nil, nil)
			So(actual, ShouldResemble, lastCheckTest)
		})
		Convey("User and time start maintenance", func() {
			actual := lastCheckTest
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, startMaintenanceUser, startMaintenanceTime)
			actual.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
			So(actual, ShouldResemble, lastCheckTest)
		})
		Convey("User and time stop maintenance", func() {
			actual := lastCheckTest
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, stopMaintenanceUser, stopMaintenanceTime)
			actual.MaintenanceInfo.Set(nil, nil, &stopMaintenanceUser, &stopMaintenanceTime)
			So(actual, ShouldResemble, lastCheckTest)
		})
		Convey("User and time start maintenance if set user and time stop maintenance", func() {
			actual := lastCheckTest
			lastCheckTest.MaintenanceInfo.Set(nil,nil, &stopMaintenanceUser, &stopMaintenanceTime)
			actual.MaintenanceInfo.Set(&startMaintenanceUser,&startMaintenanceTime, nil,nil)
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, startMaintenanceUser, startMaintenanceTime)
			So(actual, ShouldResemble, lastCheckTest)
		})
		Convey("User and time start maintenance if user is anonymous", func(){
			actual := lastCheckTest
			lastCheckTest.MaintenanceInfo.Set(nil,nil, &stopMaintenanceUser, &stopMaintenanceTime)
			SetMaintenanceUserAndTime(&lastCheckTest, triggerMaintenanceTS, "anonymous", startMaintenanceTime)
			actual.MaintenanceInfo.Set(nil,nil,nil,nil)
			So(actual, ShouldResemble, lastCheckTest)
		})
	})

	Convey("Test metric", t, func() {
		Convey("User anonymous", func(){
			actual := lastMetricsTest
			SetMaintenanceUserAndTime(&lastMetricsTest, triggerMaintenanceTS, "anonymous", startMaintenanceTime)
			actual.MaintenanceInfo.Set(nil, nil, nil, nil)
			So(actual, ShouldResemble, lastMetricsTest)
		})
		Convey("User '' ", func() {
			actual := lastMetricsTest
			SetMaintenanceUserAndTime(&lastMetricsTest, triggerMaintenanceTS, "", startMaintenanceTime)
			actual.MaintenanceInfo.Set(nil, nil, nil, nil)
			So(actual, ShouldResemble, lastMetricsTest)
		})
		Convey("User and time start maintenance", func(){
			actual := lastMetricsTest
			SetMaintenanceUserAndTime(&lastMetricsTest, triggerMaintenanceTS, startMaintenanceUser, startMaintenanceTime)
			actual.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
			So(actual, ShouldResemble, lastMetricsTest)
		})
		Convey("User and time start maintenance if user is anonymous", func(){
			actual := lastMetricsTest
			lastCheckTest.MaintenanceInfo.StopUser = &stopMaintenanceUser
			lastCheckTest.MaintenanceInfo.StopTime = &stopMaintenanceTime
			SetMaintenanceUserAndTime(&lastMetricsTest, triggerMaintenanceTS, "anonymous", startMaintenanceTime)
			actual.MaintenanceInfo.Set(nil, nil, nil, nil)
			So(actual, ShouldResemble, lastMetricsTest)
		})
	})
}

var lastCheckTest = CheckData{
	Score:     6000,
	State:     StateOK,
	Suppressed: true ,
	SuppressedState: StateERROR,
	Timestamp: 1504509981,
	Maintenance: 1000,
}

var lastMetricsTest = MetricState {
	EventTimestamp: 1504449789,
	State:          StateNODATA,
	Suppressed:     true,
	Timestamp:      1504509380,
	Maintenance:    1552723340,
}
