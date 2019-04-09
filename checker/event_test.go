package checker

import (
	"fmt"
	"testing"
	"time"

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
	Convey("Test compare metric states", t, func(c C) {
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

		Convey("Same state values", t, func(c C) {
			Convey("Status OK, no need to send", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateOK
				currentState.State = moira.StateOK

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Status NODATA and no remind interval, no need to send", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateNODATA
				currentState.State = moira.StateNODATA

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Status ERROR and no remind interval, no need to send", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateERROR
				currentState.State = moira.StateERROR

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Status NODATA and remind interval, need to send", t, func(c C) {
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
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Status ERROR and remind interval, need to send", t, func(c C) {
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
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Status EXCEPTION and lastState.Suppressed=false", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateEXCEPTION

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				c.So(actual, ShouldResemble, currentState)
			})
		})

		Convey("Test different states", t, func(c C) {
			Convey("Metric maintenance", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				currentState.Maintenance = 1502719222

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = true
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Trigger maintenance", t, func(c C) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				triggerChecker.lastCheck.Maintenance = 1502719222

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = true
				c.So(actual, ShouldResemble, currentState)
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

	Convey("Same states", t, func(c C) {
		Convey("No need send", t, func(c C) {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateOK
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			c.So(err, ShouldBeNil)
			currentCheck.EventTimestamp = lastCheck.EventTimestamp
			c.So(actual, ShouldResemble, currentCheck)
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

	Convey("Different states", t, func(c C) {
		Convey("Schedule does not allows", t, func(c C) {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateNODATA
			currentCheck.SuppressedState = lastCheck.State
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			c.So(err, ShouldBeNil)
			currentCheck.EventTimestamp = currentCheck.Timestamp
			currentCheck.Suppressed = true
			c.So(actual, ShouldResemble, currentCheck)
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
		Convey(fmt.Sprintf("Test Same Status %s after maintenance. No need to send message.", state), t, func(c C) {
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", *currentState, lastState)
			c.So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			c.So(actual, ShouldResemble, *currentState)
		})
	}
}

func TestCheckMetricStateSuppressedState(t *testing.T) {
	Convey("Test SuppressedState remembered properly", t, func(c C) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		triggerChecker := TriggerChecker{
			database:  dataBase,
			trigger:   &moira.Trigger{},
			lastCheck: &moira.CheckData{},
		}

		Convey("Test switch to maintenance. State changes OK => WARN", t, func(c C) {
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
			c.So(err, ShouldBeNil)
			currentState.Suppressed = true
			currentState.EventTimestamp = currentState.Timestamp
			c.So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes WARN => OK", t, func(c C) {
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
			c.So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = lastState.Timestamp
			c.So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes OK => ERROR", t, func(c C) {
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
			c.So(err, ShouldBeNil)
			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = currentState.Timestamp
			c.So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State didn't change", t, func(c C) {
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
			c.So(err, ShouldBeNil)
			secondState.EventTimestamp = firstState.EventTimestamp
			//secondState.SuppressedState = firstState.SuppressedState
			c.So(actual, ShouldResemble, secondState)

			Convey("Test state change state after suppressed", t, func(c C) {
				thirdState := moira.MetricState{
					Timestamp:   1800,
					Maintenance: 1500,
					State:       moira.StateERROR,
				}

				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: thirdState.Timestamp,
					State:     thirdState.State,
					OldState:  secondState.State,
					Metric:    "super.awesome.metric",
					Value:     thirdState.Value,
				}, true).Return(nil)
				actual, err = triggerChecker.compareMetricStates("super.awesome.metric", thirdState, secondState)
				c.So(err, ShouldBeNil)
				thirdState.EventTimestamp = thirdState.Timestamp
				thirdState.Suppressed = false
				c.So(actual, ShouldResemble, thirdState)
			})
		})

		Convey("Test switch out of maintenance. State changed during suppression", t, func(c C) {
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
			}

			Convey("No maintenance info", t, func(c C) {
				message := fmt.Sprintf("This metric Changed its state during maintenance interval.")
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentState.Timestamp,
					State:     currentState.State,
					OldState:  lastState.SuppressedState,
					Metric:    "super.awesome.metric",
					Value:     currentState.Value,
					Message:   &message,
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Maintenance info in metric state", t, func(c C) {
				lastState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				currentState.MaintenanceInfo = moira.MaintenanceInfo{
					StartUser: &startMetricUser,
					StartTime: &startMetricTime,
				}
				message := fmt.Sprintf("This metric Changed its state during maintenance interval. Maintenance was set by metric user at %v.", time.Unix(startMetricTime, 0).Format(format))
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentState.Timestamp,
					State:     currentState.State,
					OldState:  lastState.SuppressedState,
					Metric:    "super.awesome.metric",
					Value:     currentState.Value,
					Message:   &message,
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				c.So(actual, ShouldResemble, currentState)
			})

			Convey("Maintenance info in metric state and trigger state, but in trigger state maintenance timestamp are more", t, func(c C) {
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
				message := fmt.Sprintf("This metric Changed its state during maintenance interval. Maintenance was set by trigger user at %v.", time.Unix(startTriggerTime, 0).Format(format))
				dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
					TriggerID: triggerChecker.triggerID,
					Timestamp: currentState.Timestamp,
					State:     currentState.State,
					OldState:  lastState.SuppressedState,
					Metric:    "super.awesome.metric",
					Value:     currentState.Value,
					Message:   &message,
				}, true).Return(nil)
				actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
				c.So(err, ShouldBeNil)
				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				c.So(actual, ShouldResemble, currentState)
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

	Convey("Test trigger maintenance work properly and we don't create events", t, func(c C) {
		Convey("Compare metric state", t, func(c C) {
			Convey("No need to send", t, func(c C) {
				actual, err := triggerChecker.compareMetricStates("m1", currentMetricState, lastMetricState)
				currentMetricState.EventTimestamp = 1000
				currentMetricState.Suppressed = true
				currentMetricState.SuppressedState = moira.StateOK
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, currentMetricState)
			})

			Convey("Need to send", t, func(c C) {
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
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, currentMetricState)
			})
		})

		Convey("Compare trigger state", t, func(c C) {
			triggerChecker.lastCheck = &lastTriggerState

			Convey("No need to send", t, func(c C) {
				currentTriggerState.Maintenance = lastTriggerState.Maintenance
				actual, err := triggerChecker.compareTriggerStates(currentTriggerState)
				currentTriggerState.EventTimestamp = 1000
				currentTriggerState.Suppressed = true
				currentTriggerState.SuppressedState = moira.StateOK
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, currentTriggerState)
			})

			Convey("Need to send", t, func(c C) {
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
				c.So(err, ShouldBeNil)
				c.So(actual, ShouldResemble, currentTriggerState)
			})
		})
	})
}

func TestIsStateChanged(t *testing.T) {
	startMaintenanceUser := "testStartMtUser"
	startMaintenanceTime := int64(123)
	stopMaintenanceUser := "testStopMtUser"
	stopMaintenanceTime := int64(1230)

	Convey("isStateChanged tests", t, func(c C) {
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

		Convey("Test needSendEvents for trigger", t, func(c C) {
			Convey("Start Maintenance not start user and time", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
				needSend, message := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Start Maintenance", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"))
				lastCheckTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
				needSend, message := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Stop Maintenance", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by %v at %v and removed by %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"), stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
				lastCheckTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, &stopMaintenanceUser, &stopMaintenanceTime)
				needSend, message := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Stop Maintenance by one user", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by %v at %v and removed at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"), time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
				lastCheckTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, &startMaintenanceUser, &stopMaintenanceTime)
				needSend, message := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})

			Convey("Stop Maintenance not start user and time", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
				lastCheckTest.MaintenanceInfo.Set(nil, nil, &stopMaintenanceUser, &stopMaintenanceTime)
				needSend, message := isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, lastCheckTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
		})

		Convey("Test needSendEvents for metric", t, func(c C) {
			Convey("Start Maintenance not start user and time", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
				needSend, message := isStateChanged(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Start Maintenance", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"))
				currentMetricTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, nil, nil)
				needSend, message := isStateChanged(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Stop Maintenance", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval.")
				currentMetricTest.MaintenanceInfo.Set(nil, nil, &stopMaintenanceUser, &stopMaintenanceTime)
				needSend, message := isStateChanged(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
			Convey("Stop Maintenance not start user and time", t, func(c C) {
				actual := fmt.Sprintf("This metric changed its state during maintenance interval. Maintenance was set by %v at %v and removed by %v at %v.", startMaintenanceUser, time.Unix(startMaintenanceTime, 0).Format("15:04 02.01.2006"), stopMaintenanceUser, time.Unix(stopMaintenanceTime, 0).Format("15:04 02.01.2006"))
				currentMetricTest.MaintenanceInfo.Set(&startMaintenanceUser, &startMaintenanceTime, &stopMaintenanceUser, &stopMaintenanceTime)
				needSend, message := isStateChanged(currentMetricTest.State, lastMetricsTest.State, currentMetricTest.Timestamp, lastMetricsTest.GetEventTimestamp(), lastMetricsTest.Suppressed, lastMetricsTest.SuppressedState, currentMetricTest.MaintenanceInfo)
				c.So(needSend, ShouldBeTrue)
				c.So(*message, ShouldResemble, actual)
			})
		})
	})
}
