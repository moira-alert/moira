package checker

import (
	"fmt"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
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
			trigger:   &moira2.Trigger{},
			lastCheck: &moira2.CheckData{},
		}

		lastStateExample := moira2.MetricState{
			Timestamp:      1502712000,
			EventTimestamp: 1502708400,
			Suppressed:     false,
		}
		currentStateExample := moira2.MetricState{
			Suppressed: false,
			Timestamp:  1502719200,
			State:      moira2.StateNODATA,
		}

		Convey("Same state values", func() {
			Convey("Status OK, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira2.StateOK
				currentState.State = moira2.StateOK

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status NODATA and no remind interval, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira2.StateNODATA
				currentState.State = moira2.StateNODATA

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				So(actual, ShouldResemble, currentState)
			})

			Convey("Status ERROR and no remind interval, no need to send", func() {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira2.StateERROR
				currentState.State = moira2.StateERROR

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
				lastState.State = moira2.StateNODATA
				currentState.State = moira2.StateNODATA
				currentState.Timestamp = 1502809200

				var interval int64 = 24
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            moira2.StateNODATA,
					OldState:         moira2.StateNODATA,
					Metric:           "m1",
					Value:            currentState.Value,
					Message:          nil,
					MessageEventInfo: &moira2.EventInfo{Interval: &interval},
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
				lastState.State = moira2.StateERROR
				currentState.State = moira2.StateERROR
				currentState.Timestamp = 1502809200

				var interval int64 = 24
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            moira2.StateERROR,
					OldState:         moira2.StateERROR,
					Metric:           "m1",
					Value:            currentState.Value,
					Message:          nil,
					MessageEventInfo: &moira2.EventInfo{Interval: &interval},
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
				lastState.State = moira2.StateEXCEPTION
				currentState.State = moira2.StateEXCEPTION

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
				lastState.State = moira2.StateEXCEPTION
				currentState.State = moira2.StateOK
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
				lastState.State = moira2.StateEXCEPTION
				currentState.State = moira2.StateOK
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
		trigger:   &moira2.Trigger{},
	}

	lastCheckExample := moira2.CheckData{
		Timestamp:      1502712000,
		EventTimestamp: 1502708400,
		Suppressed:     false,
	}
	currentCheckExample := moira2.CheckData{
		Suppressed: false,
		Timestamp:  1502719200,
	}

	Convey("Same states", t, func() {
		Convey("No need send", func() {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira2.StateOK
			currentCheck.State = moira2.StateOK
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = lastCheck.EventTimestamp
			So(actual, ShouldResemble, currentCheck)
		})
	})

	triggerChecker.trigger.Schedule = &moira2.ScheduleData{
		TimezoneOffset: -300,
		StartOffset:    0,
		EndOffset:      1439,
		Days: []moira2.ScheduleDataDay{
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
			lastCheck.State = moira2.StateOK
			currentCheck.State = moira2.StateNODATA
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
		trigger:   &moira2.Trigger{},
		lastCheck: &moira2.CheckData{},
	}

	lastState := moira2.MetricState{
		Timestamp:      1000,
		EventTimestamp: 1,
		Suppressed:     true,
		Maintenance:    1100,
		State:          moira2.StateWARN,
	}
	currentState := newMetricState(lastState, moira2.StateWARN, 1200, nil)
	states := []moira2.State{moira2.StateOK, moira2.StateWARN, moira2.StateERROR, moira2.StateNODATA, moira2.StateEXCEPTION}

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
			trigger:   &moira2.Trigger{},
			lastCheck: &moira2.CheckData{},
		}

		Convey("Test switch to maintenance. State changes OK => WARN", func() {
			lastState := moira2.MetricState{
				Timestamp:      100,
				EventTimestamp: 10,
				State:          moira2.StateOK,
			}
			currentState := moira2.MetricState{
				Timestamp:   1000,
				Maintenance: 1500,
				State:       moira2.StateWARN,
			}

			currentState.SuppressedState = lastState.State
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.Suppressed = true
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes WARN => OK", func() {
			lastState := moira2.MetricState{
				Timestamp:       1000,
				EventTimestamp:  1000,
				Maintenance:     1500,
				State:           moira2.StateWARN,
				Suppressed:      true,
				SuppressedState: moira2.StateOK,
			}

			currentState := moira2.MetricState{
				Timestamp:   1100,
				Maintenance: 1500,
				State:       moira2.StateOK,
				Suppressed:  true,
			}
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = lastState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes OK => ERROR", func() {
			lastState := moira2.MetricState{
				Timestamp:       1100,
				EventTimestamp:  1000,
				Maintenance:     1500,
				State:           moira2.StateOK,
				Suppressed:      true,
				SuppressedState: moira2.StateOK,
			}

			currentState := moira2.MetricState{
				Timestamp:   1200,
				Maintenance: 1500,
				State:       moira2.StateERROR,
				Suppressed:  true,
			}

			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State didn't change", func() {
			firstState := moira2.MetricState{
				Timestamp:       1200,
				EventTimestamp:  1200,
				Maintenance:     1500,
				State:           moira2.StateERROR,
				Suppressed:      true,
				SuppressedState: moira2.StateOK,
			}

			secondState := moira2.MetricState{
				Timestamp:   1600,
				Maintenance: 1500,
				State:       moira2.StateOK,
			}

			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", secondState, firstState)
			So(err, ShouldBeNil)
			secondState.EventTimestamp = firstState.EventTimestamp
			//secondState.SuppressedState = firstState.SuppressedState
			So(actual, ShouldResemble, secondState)

			Convey("Test state change state after suppressed", func() {
				thirdState := moira2.MetricState{
					Timestamp:   1800,
					Maintenance: 1500,
					State:       moira2.StateERROR,
				}

				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: thirdState.Timestamp,
					State:     thirdState.State,
					OldState:  secondState.State,
					Metric:    "super.awesome.metric",
					Value:     thirdState.Value,
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

			lastState := moira2.MetricState{
				Timestamp:       1200,
				EventTimestamp:  1200,
				Maintenance:     1500,
				State:           moira2.StateOK,
				Suppressed:      true,
				SuppressedState: moira2.StateOK,
			}

			currentState := moira2.MetricState{
				Timestamp:   1600,
				Maintenance: 1500,
				State:       moira2.StateERROR,
				Suppressed:  true,
			}

			Convey("No maintenance info", func() {
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Value:            currentState.Value,
					MessageEventInfo: &moira2.EventInfo{Maintenance: &moira2.MaintenanceInfo{}},
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				So(actual, ShouldResemble, currentState)
			})

			Convey("Maintenance info in metric state", func() {
				lastState.MaintenanceInfo = moira2.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				currentState.MaintenanceInfo = moira2.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Value:            currentState.Value,
					MessageEventInfo: &moira2.EventInfo{Maintenance: &lastState.MaintenanceInfo},
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
				triggerChecker.lastCheck.MaintenanceInfo = moira2.MaintenanceInfo{
					StartUser: &startTriggerUser,
					StartTime: &startTriggerTime,
				}
				lastState.MaintenanceInfo = moira2.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				currentState.MaintenanceInfo = moira2.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:        triggerChecker.triggerID,
					Timestamp:        currentState.Timestamp,
					State:            currentState.State,
					OldState:         lastState.SuppressedState,
					Metric:           "super.awesome.metric",
					Value:            currentState.Value,
					MessageEventInfo: &moira2.EventInfo{Maintenance: &triggerChecker.lastCheck.MaintenanceInfo},
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
		trigger: &moira2.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
		lastCheck: &moira2.CheckData{
			Maintenance: 1500,
		},
	}

	lastMetricState := moira2.MetricState{
		Timestamp:      100,
		EventTimestamp: 10,
		State:          moira2.StateOK,
	}
	currentMetricState := moira2.MetricState{
		Timestamp: 1000,
		State:     moira2.StateWARN,
	}

	lastTriggerState := moira2.CheckData{
		Maintenance:    1500,
		Timestamp:      100,
		EventTimestamp: 10,
		Suppressed:     false,
		State:          moira2.StateOK,
	}

	currentTriggerState := moira2.CheckData{
		Timestamp:  1000,
		Suppressed: false,
		State:      moira2.StateERROR,
	}

	Convey("Test trigger maintenance work properly and we don't create events", t, func() {
		Convey("Compare metric state", func() {
			Convey("No need to send", func() {
				actual, err := triggerChecker.compareMetricStates("m1", currentMetricState, lastMetricState)
				currentMetricState.EventTimestamp = 1000
				currentMetricState.Suppressed = true
				currentMetricState.SuppressedState = moira2.StateOK
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentMetricState)
			})

			Convey("Need to send", func() {
				currentMetricState.Timestamp = 1600
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentMetricState.Timestamp,
					State:     moira2.StateWARN,
					OldState:  moira2.StateOK,
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
				currentTriggerState.SuppressedState = moira2.StateOK
				So(err, ShouldBeNil)
				So(actual, ShouldResemble, currentTriggerState)
			})

			Convey("Need to send", func() {
				currentTriggerState.Timestamp = 1600
				dataBase.EXPECT().PushNotificationEvent(&moira2.NotificationEvent{
					TriggerID:      triggerChecker.triggerID,
					Timestamp:      currentTriggerState.Timestamp,
					State:          moira2.StateERROR,
					OldState:       moira2.StateOK,
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
		var lastCheckTest = moira2.CheckData{
			Score:           6000,
			State:           moira2.StateOK,
			Suppressed:      true,
			SuppressedState: moira2.StateERROR,
			Timestamp:       1504509981,
			Maintenance:     1000,
		}

		var currentCheckTest = moira2.CheckData{
			State:     moira2.StateWARN,
			Timestamp: 1504509981,
		}

		Convey("Test is state changed", func() {
			Convey("If is last check suppressed and current state not equal last state", func() {
				lastCheckTest.Suppressed = false
				eventInfo, needSend := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-1, lastCheckTest.Suppressed, lastCheckTest.SuppressedState, moira2.MaintenanceInfo{})
				So(eventInfo, ShouldBeNil)
				So(needSend, ShouldBeTrue)
			})

			Convey("Create EventInfo with MaintenanceInfo", func() {
				maintenanceInfo := moira2.MaintenanceInfo{}
				eventInfo, needSend := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, maintenanceInfo)
				So(eventInfo, ShouldNotBeNil)
				So(eventInfo, ShouldResemble, &moira2.EventInfo{Maintenance: &maintenanceInfo})
				So(needSend, ShouldBeTrue)
			})

			Convey("Create EventInfo with interval", func() {
				var interval int64 = 24
				eventInfo, needSend := isStateChanged(moira2.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-100000, lastCheckTest.Suppressed, moira2.StateNODATA, moira2.MaintenanceInfo{})
				So(eventInfo, ShouldNotBeNil)
				So(eventInfo, ShouldResemble, &moira2.EventInfo{Interval: &interval})
				So(needSend, ShouldBeTrue)
			})

			Convey("No send message", func() {
				eventInfo, needSend := isStateChanged(moira2.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, moira2.StateNODATA, moira2.MaintenanceInfo{})
				So(eventInfo, ShouldBeNil)
				So(needSend, ShouldBeFalse)
			})
		})
	})
}
