package checker

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
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

				currentState.Values = map[string]float64{"t1": 0}
				var interval int64 = 24
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            moira.StateNODATA,
					OldState:         moira.StateNODATA,
					Metric:           "m1",
					Values:           map[string]float64{"t1": 0},
					Message:          nil,
					MessageEventInfo: &moira.EventInfo{Interval: &interval},
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
				currentState.Values = map[string]float64{"t1": 0}

				var interval int64 = 24
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            moira.StateERROR,
					OldState:         moira.StateERROR,
					Metric:           "m1",
					Values:           map[string]float64{"t1": 0},
					Message:          nil,
					MessageEventInfo: &moira.EventInfo{Interval: &interval},
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
			Convey("Metric maintenance", func() {
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

			Convey("Trigger maintenance", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				triggerChecker.lastCheck.Maintenance = 1502719222

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
	triggerChecker := TriggerChecker{
		trigger:   &moira.Trigger{},
		lastCheck: &moira.CheckData{},
	}

	lastState := moira.MetricState{
		Timestamp:      1000,
		EventTimestamp: 1,
		Suppressed:     true,
		Maintenance:    1100,
		State:          moira.StateWARN,
	}
	currentState := newMetricState(lastState, moira.StateWARN, 1200, nil)
	states := []moira.State{moira.StateOK, moira.StateWARN, moira.StateERROR, moira.StateNODATA, moira.StateEXCEPTION}

	for _, state := range states {
		lastState.State = state
		currentState.State = state
		Convey(fmt.Sprintf("Test Same Status %s after maintenance. No need to send message.", state), t, func() {
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", *currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, *currentState)
		})
	}
}

func TestCheckMetricStateSuppressedState(t *testing.T) {
	Convey("Test SuppressedState remembered properly", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		triggerChecker := TriggerChecker{
			database:  dataBase,
			trigger:   &moira.Trigger{},
			lastCheck: &moira.CheckData{},
		}

		Convey("Test switch to maintenance. State changes OK => WARN", func() {
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

			currentState.SuppressedState = lastState.State
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.Suppressed = true
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes WARN => OK", func() {
			lastState := moira.MetricState{
				Timestamp:       1000,
				EventTimestamp:  1000,
				Maintenance:     1500,
				State:           moira.StateWARN,
				Suppressed:      true,
				SuppressedState: moira.StateOK,
			}

			currentState := moira.MetricState{
				Timestamp:   1100,
				Maintenance: 1500,
				State:       moira.StateOK,
				Suppressed:  true,
			}
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = lastState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes OK => ERROR", func() {
			lastState := moira.MetricState{
				Timestamp:       1100,
				EventTimestamp:  1000,
				Maintenance:     1500,
				State:           moira.StateOK,
				Suppressed:      true,
				SuppressedState: moira.StateOK,
			}

			currentState := moira.MetricState{
				Timestamp:   1200,
				Maintenance: 1500,
				State:       moira.StateERROR,
				Suppressed:  true,
			}

			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State didn't change", func() {
			firstState := moira.MetricState{
				Timestamp:       1200,
				EventTimestamp:  1200,
				Maintenance:     1500,
				State:           moira.StateERROR,
				Suppressed:      true,
				SuppressedState: moira.StateOK,
			}

			secondState := moira.MetricState{
				Timestamp:   1600,
				Maintenance: 1500,
				State:       moira.StateOK,
			}

			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", secondState, firstState)
			So(err, ShouldBeNil)
			secondState.EventTimestamp = firstState.EventTimestamp
			//secondState.SuppressedState = firstState.SuppressedState
			So(actual, ShouldResemble, secondState)

			Convey("Test state change state after suppressed", func() {
				thirdState := moira.MetricState{
					Timestamp:   1800,
					Maintenance: 1500,
					State:       moira.StateERROR,
					Values:      map[string]float64{"t1": 0},
				}

				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: thirdState.Timestamp,
					State:     thirdState.State,
					OldState:  secondState.State,
					Metric:    "super.awesome.metric",
					Values:    map[string]float64{"t1": 0},
				}, true).Return(nil)
				actual, err = triggerChecker.compareMetricStates("super.awesome.metric", thirdState, secondState)
				So(err, ShouldBeNil)
				thirdState.EventTimestamp = thirdState.Timestamp
				thirdState.Suppressed = false
				So(actual, ShouldResemble, thirdState)
			})
		})

		Convey("Test switch out of maintenance. State changed during suppression", func() {
			startMetricUser := "metric user"
			startMetricTime := int64(900)
			startTriggerUser := "trigger user"
			startTriggerTime := int64(1000)

			lastState := moira.MetricState{
				Timestamp:       1200,
				EventTimestamp:  1200,
				Maintenance:     1500,
				State:           moira.StateOK,
				Suppressed:      true,
				SuppressedState: moira.StateOK,
			}

			currentState := moira.MetricState{
				Timestamp:   1600,
				Maintenance: 1500,
				State:       moira.StateERROR,
				Suppressed:  true,
				Values:      map[string]float64{"t1": 0},
			}

			Convey("No maintenance info", func() {
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Values:           map[string]float64{"t1": 0},
					MessageEventInfo: &moira.EventInfo{Maintenance: &moira.MaintenanceInfo{}},
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				So(actual, ShouldResemble, currentState)
			})

			Convey("Maintenance info in metric state", func() {
				lastState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				currentState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Values:           map[string]float64{"t1": 0},
					MessageEventInfo: &moira.EventInfo{Maintenance: &lastState.MaintenanceInfo},
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				So(actual, ShouldResemble, currentState)
			})

			Convey("Maintenance info in metric state and trigger state, but in trigger state maintenance timestamp are more", func() {
				triggerChecker.lastCheck.Maintenance = 1550
				triggerChecker.lastCheck.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startTriggerUser,
					StartTime: &startTriggerTime,
				}
				lastState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				currentState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Values:           map[string]float64{"t1": 0},
					MessageEventInfo: &moira.EventInfo{Maintenance: &triggerChecker.lastCheck.MaintenanceInfo},
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				So(actual, ShouldResemble, currentState)
			})
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
		Values:    map[string]float64{"t1": 0},
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
					Values:    map[string]float64{"t1": 0},
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
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID:      triggerChecker.triggerID,
					Timestamp:      currentTriggerState.Timestamp,
					State:          moira.StateERROR,
					OldState:       moira.StateOK,
					Metric:         "",
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

func TestIsStateChanged(t *testing.T) {
	Convey("isStateChanged tests", t, func() {
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

		Convey("Test is state changed", func() {
			Convey("If is last check suppressed and current state not equal last state", func() {
				lastCheckTest.Suppressed = false
				eventInfo, needSend := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-1, lastCheckTest.Suppressed, lastCheckTest.SuppressedState, moira.MaintenanceInfo{})
				So(eventInfo, ShouldBeNil)
				So(needSend, ShouldBeTrue)
			})

			Convey("Create EventInfo with MaintenanceInfo", func() {
				maintenanceInfo := moira.MaintenanceInfo{}
				eventInfo, needSend := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, maintenanceInfo)
				So(eventInfo, ShouldNotBeNil)
				So(eventInfo, ShouldResemble, &moira.EventInfo{Maintenance: &maintenanceInfo})
				So(needSend, ShouldBeTrue)
			})

			Convey("Create EventInfo with interval", func() {
				var interval int64 = 24
				eventInfo, needSend := isStateChanged(moira.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-100000, lastCheckTest.Suppressed, moira.StateNODATA, moira.MaintenanceInfo{})
				So(eventInfo, ShouldNotBeNil)
				So(eventInfo, ShouldResemble, &moira.EventInfo{Interval: &interval})
				So(needSend, ShouldBeTrue)
			})

			Convey("No send message", func() {
				eventInfo, needSend := isStateChanged(moira.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, moira.StateNODATA, moira.MaintenanceInfo{})
				So(eventInfo, ShouldBeNil)
				So(needSend, ShouldBeFalse)
			})
		})
	})
}
