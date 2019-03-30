package checker

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func newMocks(t *testing.T) (dataBase *mock_moira_alert.MockDatabase, mockCtrl *gomock.Controller) {
	mockCtrl = gomock.NewController(t)
	dataBase = mock_moira_alert.NewMockDatabase(mockCtrl)
	return
}

func TestCompareMetricStates(t *testing.T) {
	Convey("Test compare metric states", t, func() {
		logger, _ := logging.GetLogger("Test")

		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			logger:    logger,
			trigger:   &moira.Trigger{},
			lastCheck: &moira.CheckData{},
		}

		lastStateExample := moira.MetricState{
			Timestamp:      1502712000,
			EventTimestamp: 1502708400,
			Suppressed:     false,
		}
		currentStateExample := moira.MetricState{
			Suppressed: false,
			Timestamp:  1502719200,
			State:      moira.StateNODATA,
		}

		Convey("Same state values", func() {
			Convey("Status OK, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateOK
				currentState.State = moira.StateOK

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status NODATA and no remind interval, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateNODATA
				currentState.State = moira.StateNODATA

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status ERROR and no remind interval, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateERROR
				currentState.State = moira.StateERROR

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status NODATA and remind interval, need to send", func() {
				dataBase, mockCtrl := newMocks(t)
				triggerChecker.database = dataBase
				defer mockCtrl.Finish()

				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateNODATA
				currentState.State = moira.StateNODATA
				currentState.Timestamp = 1502809200

				message := fmt.Sprintf(remindMessage, 24)
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentState.Timestamp,
					State:     moira.StateNODATA,
					OldState:  moira.StateNODATA,
					Metric:    "m1",
					Value:     currentState.Value,
					Message:   &message,
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status ERROR and remind interval, need to send", func() {
				dataBase, mockCtrl := newMocks(t)
				triggerChecker.database = dataBase
				defer mockCtrl.Finish()

				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateERROR
				currentState.State = moira.StateERROR
				currentState.Timestamp = 1502809200

				message := fmt.Sprintf(remindMessage, 24)
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentState.Timestamp,
					State:     moira.StateERROR,
					OldState:  moira.StateERROR,
					Metric:    "m1",
					Value:     currentState.Value,
					Message:   &message,
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status EXCEPTION and lastState.Suppressed=false", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateEXCEPTION

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})
		})

		Convey("Test different states", func() {
			Convey("Trigger maintenance", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				currentState.Maintenance = 1502719222

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = true
				So(actual, ShouldResemble, currentState)
			})
		})
	})
}

func TestCompareTriggerStates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		triggerID: "SuperId",
		database:  dataBase,
		logger:    logger,
		trigger:   &moira.Trigger{},
	}

	lastCheckExample := moira.CheckData{
		Timestamp:      1502712000,
		EventTimestamp: 1502708400,
		Suppressed:     false,
	}
	currentCheckExample := moira.CheckData{
		Suppressed: false,
		Timestamp:  1502719200,
	}

	Convey("Same states", t, func() {
		Convey("No need send", func() {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateOK
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = lastCheck.EventTimestamp
			So(actual, ShouldResemble, currentCheck)
		})
	})

	triggerChecker.trigger.Schedule = &moira.ScheduleData{
		TimezoneOffset: -300,
		StartOffset:    0,
		EndOffset:      1439,
		Days: []moira.ScheduleDataDay{
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

	Convey("Different states", t, func() {
		Convey("Schedule does not allows", func() {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateNODATA
			currentCheck.SuppressedState = lastCheck.State
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = currentCheck.Timestamp
			currentCheck.Suppressed = true
			So(actual, ShouldResemble, currentCheck)
		})
	})
}

func TestCheckMetricStateWithLastStateSuppressed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		logger:   logger,
		until:    67,
		from:     17,
		database: dataBase,
		trigger: &moira.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
		lastCheck: &moira.CheckData{},
	}

	lastState := moira.MetricState{
		Timestamp:      1000,
		EventTimestamp: 1,
		Suppressed:     true,
		Maintenance:    1100,
		State:          moira.StateWARN,
	}
	currentState := moira.MetricState{
		Suppressed: false,
		Timestamp:  1200,
		State:      moira.StateWARN,
	}

	states := []moira.State{moira.StateOK, moira.StateWARN, moira.StateERROR, moira.StateNODATA, moira.StateEXCEPTION}

	for _, state := range states {
		Convey(fmt.Sprintf("Test Same Status %s after maintenance. No need to send message.", state), t, func() {
			lastState.State = state
			currentState.State = state
			currentState.SuppressedState = lastState.State
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			So(actual, ShouldResemble, currentState)
		})
	}
}

func TestCheckMetricStateSuppressedState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		logger:   logger,
		until:    67,
		from:     17,
		database: dataBase,
		trigger: &moira.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
		lastCheck: &moira.CheckData{},
	}

	lastState := moira.MetricState{
		Timestamp:      100,
		EventTimestamp: 10,
		State:          moira.StateOK,
	}
	currentState := moira.MetricState{
		Timestamp:   1000,
		Maintenance: 1500,
		State:       moira.StateWARN,
	}

	Convey("Test SuppressedState remembered properly", t, func() {
		Convey("Test switch to maintenance. State changes OK => WARN", func() {
			currentState.SuppressedState = lastState.State
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.Suppressed = true
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes WARN => OK", func() {
			lastState = currentState
			currentState.Timestamp = 1100
			currentState.State = moira.StateOK
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes OK => ERROR", func() {
			lastState = currentState
			currentState.Timestamp = 1200
			currentState.State = moira.StateERROR
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State didn't change", func() {
			lastState = currentState
			currentState.Timestamp = 1600
			currentState.State = moira.StateOK
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State changed during suppression", func() {
			lastState = moira.MetricState{
				EventTimestamp:  1300,
				Timestamp:       1300,
				Maintenance:     1500,
				Suppressed:      true,
				SuppressedState: moira.StateOK,
				State:           moira.StateERROR,
			}
			currentState.Timestamp = 1600
			currentState.State = moira.StateERROR

			message := fmt.Sprintf("This metric changed its state during maintenance interval.")
			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				TriggerID: triggerChecker.triggerID,
				Timestamp: currentState.Timestamp,
				State:     currentState.State,
				OldState:  currentState.SuppressedState,
				Metric:    "super.awesome.metric",
				Value:     currentState.Value,
				Message:   &message,
			}, true).Return(nil)

			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.Suppressed = false
			currentState.SuppressedState = ""
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})
	})
}

func TestTriggerMaintenance(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		logger:   logger,
		until:    67,
		from:     17,
		database: dataBase,
		trigger: &moira.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
		lastCheck: &moira.CheckData{
			Maintenance: 1500,
		},
	}

	lastMetricState := moira.MetricState{
		Timestamp:      100,
		EventTimestamp: 10,
		State:          moira.StateOK,
	}
	currentMetricState := moira.MetricState{
		Timestamp: 1000,
		State:     moira.StateWARN,
	}

	lastTriggerState := moira.CheckData{
		Maintenance:    1500,
		Timestamp:      100,
		EventTimestamp: 10,
		Suppressed:     false,
		State:          moira.StateOK,
	}

	currentTriggerState := moira.CheckData{
		Timestamp:  1000,
		Suppressed: false,
		State:      moira.StateERROR,
	}

	Convey("Test trigger maintenance work properly and we don't create events", t, func() {
		Convey("Compare metric state", func() {

			Convey("No need to send", func() {
				actual, err := triggerChecker.compareMetricStates("m1", currentMetricState, lastMetricState)
				currentMetricState.EventTimestamp = 1000
				currentMetricState.Suppressed = true
				currentMetricState.SuppressedState = moira.StateOK
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentMetricState)
			})

			Convey("Need to send", func() {
				currentMetricState.Timestamp = 1600
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentMetricState.Timestamp,
					State:     moira.StateWARN,
					OldState:  moira.StateOK,
					Metric:    "m1",
					Value:     currentMetricState.Value,
				}, true).Return(nil)

				actual, err := triggerChecker.compareMetricStates("m1", currentMetricState, lastMetricState)
				currentMetricState.EventTimestamp = 1600
				currentMetricState.Suppressed = false
				currentMetricState.SuppressedState = ""
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentMetricState)
			})
		})

		Convey("Compare trigger state", func() {
			triggerChecker.lastCheck = &lastTriggerState

			Convey("No need to send", func() {
				currentTriggerState.Maintenance = lastTriggerState.Maintenance
				actual, err := triggerChecker.compareTriggerStates(currentTriggerState)
				currentTriggerState.EventTimestamp = 1000
				currentTriggerState.Suppressed = true
				currentTriggerState.SuppressedState = moira.StateOK
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentTriggerState)
			})

			Convey("Need to send", func() {
				currentTriggerState.Timestamp = 1600
				emptyEvent := ""
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:      triggerChecker.triggerID,
					Timestamp:      currentTriggerState.Timestamp,
					State:          moira.StateERROR,
					OldState:       moira.StateOK,
					Metric:         "",
					Message:        &emptyEvent,
					IsTriggerEvent: true,
				}, true).Return(nil)

				actual, err := triggerChecker.compareTriggerStates(currentTriggerState)
				currentTriggerState.EventTimestamp = 1600
				currentTriggerState.Suppressed = false
				currentTriggerState.SuppressedState = ""
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentTriggerState)
			})
		})
	})
}

func TestNeedSendEvent(t *testing.T) {
	startMaintenanceUser := "testStartMtUser"
	startMaintenanceTime := int64(123)
	stopMaintenanceUser := "testStopMtUser"
	stopMaintenanceTime := int64(1230)

	Convey("Test needSendEvents for trigger", t, func() {
		Convey("Start Maintenance not start user and time", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
			needSend, message := needSendEvent(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Start Maintenance", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by user %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"))
			lastCheckTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
			needSend, message := needSendEvent(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Stop Maintenance", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by user %v at %v. Maintenance removed by user %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"), stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
			lastCheckTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, &stopMaintenanceUser, &stopMaintenanceTime)
			needSend, message := needSendEvent(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Stop Maintenance not start user and time", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance removed by user %v at %v.", stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
			lastCheckTest.MaintenanceInfo.Set(nil, nil, &stopMaintenanceUser, &stopMaintenanceTime)
			needSend, message := needSendEvent(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
	})

	Convey("Test needSendEvents for metric", t, func() {
		Convey("Start Maintenance not start user and time", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
			needSend, message := needSendEvent(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Start Maintenance", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by user %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"))
			currentMetricTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
			needSend, message := needSendEvent(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Stop Maintenance", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance removed by user %v at %v.", stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
			currentMetricTest.MaintenanceInfo.Set(nil, nil, &stopMaintenanceUser, &stopMaintenanceTime)
			needSend, message := needSendEvent(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
		Convey("Stop Maintenance not start user and time", func() {
			actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by user %v at %v. Maintenance removed by user %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"), stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
			currentMetricTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, &stopMaintenanceUser, &stopMaintenanceTime)
			needSend, message := needSendEvent(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
			So(needSend, ShouldBeTrue)
			So(*message, ShouldResemble, actual)
		})
	})
}

var lastCheckTest = moira.CheckData{
	Score:           6000,
	State:           moira.StateOK,
	Suppressed:      true,
	SuppressedState: moira.StateERROR,
	Timestamp:       1504509981,
	Maintenance:     1000,
}

var currentCheckTest = moira.CheckData{
	State:     moira.StateWARN,
	Timestamp: 1504509981,
}

var currentMetricTest = moira.MetricState{
	EventTimestamp: 1504449789,
	State:          moira.StateWARN,
	Suppressed:     true,
	Timestamp:      1504509380,
}

var lastMetricsTest = moira.MetricState{
	EventTimestamp: 1504449789,
	State:          moira.StateNODATA,
	Suppressed:     true,
	Timestamp:      1504509380,
	Maintenance:    1552723340,
}
