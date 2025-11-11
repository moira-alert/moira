package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"go.uber.org/mock/gomock"
)

func newMocks(t *testing.T) (dataBase *mock_moira_alert.MockDatabase, mockCtrl *gomock.Controller) {
	mockCtrl = gomock.NewController(t)
	dataBase = mock_moira_alert.NewMockDatabase(mockCtrl)

	return dataBase, mockCtrl
}

func TestCompareMetricStates(t *testing.T) {
	t.Run("Test compare metric states", func(t *testing.T) {
		logger, _ := logging.GetLogger("Test")

		triggerChecker := TriggerChecker{
			triggerID: "SuperId",
			logger:    logger,
			trigger:   &moira.Trigger{},
			lastCheck: &moira.CheckData{},
			badStateReminder: map[moira.State]int64{
				moira.StateERROR:     moira.DefaultBadStateReminder,
				moira.StateNODATA:    moira.DefaultBadStateReminder,
				moira.StateEXCEPTION: moira.DefaultBadStateReminder,
			},
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

		t.Run("Same state values", func(t *testing.T) {
			t.Run("Status OK, no need to send", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateOK
				currentState.State = moira.StateOK

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})

			t.Run("Status NODATA and no remind interval, no need to send", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateNODATA
				currentState.State = moira.StateNODATA

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})

			t.Run("Status ERROR and no remind interval, no need to send", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateERROR
				currentState.State = moira.StateERROR

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})

			t.Run("Status NODATA and remind interval, need to send", func(t *testing.T) {
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
				require.NoError(t, err)

				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})

			t.Run("Status ERROR and remind interval, need to send", func(t *testing.T) {
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
				require.NoError(t, err)

				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})

			t.Run("Status EXCEPTION and lastState.Suppressed=false", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateEXCEPTION

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = lastState.EventTimestamp
				currentState.Suppressed = false
				require.Equal(t, currentState, actual)
			})
		})

		t.Run("Test different states", func(t *testing.T) {
			t.Run("Metric maintenance", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				currentState.Maintenance = 1502719222

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = true
				require.Equal(t, currentState, actual)
			})

			t.Run("Trigger maintenance", func(t *testing.T) {
				lastState := lastStateExample
				currentState := currentStateExample
				lastState.State = moira.StateEXCEPTION
				currentState.State = moira.StateOK
				currentState.SuppressedState = lastState.State
				triggerChecker.lastCheck.Maintenance = 1502719222

				actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
				require.NoError(t, err)

				currentState.EventTimestamp = currentState.Timestamp
				currentState.Suppressed = true
				require.Equal(t, currentState, actual)
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
		badStateReminder: map[moira.State]int64{
			moira.StateERROR:     moira.DefaultBadStateReminder,
			moira.StateNODATA:    moira.DefaultBadStateReminder,
			moira.StateEXCEPTION: moira.DefaultBadStateReminder,
		},
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

	t.Run("Same states", func(t *testing.T) {
		t.Run("No need send", func(t *testing.T) {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateOK
			actual, err := triggerChecker.compareTriggerStates(currentCheck)
			require.NoError(t, err)

			currentCheck.EventTimestamp = lastCheck.EventTimestamp
			require.Equal(t, currentCheck, actual)
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

	t.Run("Different states", func(t *testing.T) {
		t.Run("Schedule does not allows", func(t *testing.T) {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = moira.StateOK
			currentCheck.State = moira.StateNODATA
			currentCheck.SuppressedState = lastCheck.State
			actual, err := triggerChecker.compareTriggerStates(currentCheck)

			require.NoError(t, err)

			currentCheck.EventTimestamp = currentCheck.Timestamp
			currentCheck.Suppressed = true
			require.Equal(t, currentCheck, actual)
		})
	})
}

func TestCheckMetricStateWithLastStateSuppressed(t *testing.T) {
	triggerChecker := TriggerChecker{
		trigger:   &moira.Trigger{},
		lastCheck: &moira.CheckData{},
		badStateReminder: map[moira.State]int64{
			moira.StateERROR:     moira.DefaultBadStateReminder,
			moira.StateNODATA:    moira.DefaultBadStateReminder,
			moira.StateEXCEPTION: moira.DefaultBadStateReminder,
		},
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
		t.Run(fmt.Sprintf("Test Same Status %s after maintenance. No need to send message.", state), func(t *testing.T) {
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", *currentState, lastState)
			require.NoError(t, err)

			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			require.Equal(t, *currentState, actual)
		})
	}
}

func TestCheckMetricStateSuppressedState(t *testing.T) {
	t.Run("Test SuppressedState remembered properly", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		triggerChecker := TriggerChecker{
			database:  dataBase,
			trigger:   &moira.Trigger{},
			lastCheck: &moira.CheckData{},
			badStateReminder: map[moira.State]int64{
				moira.StateERROR:     moira.DefaultBadStateReminder,
				moira.StateNODATA:    moira.DefaultBadStateReminder,
				moira.StateEXCEPTION: moira.DefaultBadStateReminder,
			},
		}

		t.Run("Test switch to maintenance. State changes OK => WARN", func(t *testing.T) {
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
			require.NoError(t, err)

			currentState.Suppressed = true
			currentState.EventTimestamp = currentState.Timestamp
			require.Equal(t, currentState, actual)
		})

		t.Run("Test still in maintenance. State changes WARN => OK", func(t *testing.T) {
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
			require.NoError(t, err)

			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = lastState.Timestamp
			require.Equal(t, currentState, actual)
		})

		t.Run("Test still in maintenance. State changes OK => ERROR", func(t *testing.T) {
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
			require.NoError(t, err)

			currentState.SuppressedState = lastState.SuppressedState
			currentState.EventTimestamp = currentState.Timestamp
			require.Equal(t, currentState, actual)
		})

		t.Run("Test switch out of maintenance. State didn't change", func(t *testing.T) {
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
			require.NoError(t, err)

			secondState.EventTimestamp = firstState.EventTimestamp
			require.Equal(t, secondState, actual)

			t.Run("Test state change state after suppressed", func(t *testing.T) {
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
				require.NoError(t, err)

				thirdState.EventTimestamp = thirdState.Timestamp
				thirdState.Suppressed = false
				require.Equal(t, thirdState, actual)
			})
		})

		t.Run("Test switch out of maintenance. State changed during suppression", func(t *testing.T) {
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

			t.Run("No maintenance info", func(t *testing.T) {
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
				require.NoError(t, err)

				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				require.Equal(t, currentState, actual)
			})

			t.Run("Maintenance info in metric state", func(t *testing.T) {
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
				require.NoError(t, err)

				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				require.Equal(t, currentState, actual)
			})

			t.Run("Maintenance info in metric state and trigger state, but in trigger state maintenance timestamp are more", func(t *testing.T) {
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
				require.NoError(t, err)

				currentState.Suppressed = false
				currentState.SuppressedState = ""
				currentState.EventTimestamp = currentState.Timestamp
				require.Equal(t, currentState, actual)
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
		badStateReminder: map[moira.State]int64{
			moira.StateERROR:     moira.DefaultBadStateReminder,
			moira.StateNODATA:    moira.DefaultBadStateReminder,
			moira.StateEXCEPTION: moira.DefaultBadStateReminder,
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

	t.Run("Test trigger maintenance work properly and we don't create events", func(t *testing.T) {
		t.Run("Compare metric state", func(t *testing.T) {
			t.Run("No need to send", func(t *testing.T) {
				actual, err := triggerChecker.compareMetricStates("m1", currentMetricState, lastMetricState)
				currentMetricState.EventTimestamp = 1000
				currentMetricState.Suppressed = true
				currentMetricState.SuppressedState = moira.StateOK

				require.NoError(t, err)
				require.Equal(t, currentMetricState, actual)
			})

			t.Run("Need to send", func(t *testing.T) {
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

				require.NoError(t, err)
				require.Equal(t, currentMetricState, actual)
			})
		})

		t.Run("Compare trigger state", func(t *testing.T) {
			triggerChecker.lastCheck = &lastTriggerState

			t.Run("No need to send", func(t *testing.T) {
				currentTriggerState.Maintenance = lastTriggerState.Maintenance
				actual, err := triggerChecker.compareTriggerStates(currentTriggerState)
				currentTriggerState.EventTimestamp = 1000
				currentTriggerState.Suppressed = true
				currentTriggerState.SuppressedState = moira.StateOK

				require.NoError(t, err)
				require.Equal(t, currentTriggerState, actual)
			})

			t.Run("Need to send", func(t *testing.T) {
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

				require.NoError(t, err)
				require.Equal(t, currentTriggerState, actual)
			})
		})
	})
}

func TestIsStateChanged(t *testing.T) {
	t.Run("isStateChanged tests", func(t *testing.T) {
		lastCheckTest := moira.CheckData{
			Score:           6000,
			State:           moira.StateOK,
			Suppressed:      true,
			SuppressedState: moira.StateERROR,
			Timestamp:       1504509981,
			Maintenance:     1000,
		}

		currentCheckTest := moira.CheckData{
			State:     moira.StateWARN,
			Timestamp: 1504509981,
		}

		triggerChecker := &TriggerChecker{
			badStateReminder: map[moira.State]int64{
				moira.StateERROR:     moira.DefaultBadStateReminder,
				moira.StateNODATA:    moira.DefaultBadStateReminder,
				moira.StateEXCEPTION: moira.DefaultBadStateReminder,
			},
		}

		t.Run("Test is state changed", func(t *testing.T) {
			t.Run("If is last check suppressed and current state not equal last state", func(t *testing.T) {
				lastCheckTest.Suppressed = false
				eventInfo, needSend := triggerChecker.isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-1, lastCheckTest.Suppressed, lastCheckTest.SuppressedState, moira.MaintenanceInfo{})
				require.Nil(t, eventInfo)
				require.True(t, needSend)

				lastCheckTest.Suppressed = true
			})

			t.Run("Create EventInfo with MaintenanceInfo", func(t *testing.T) {
				maintenanceInfo := moira.MaintenanceInfo{}
				eventInfo, needSend := triggerChecker.isStateChanged(currentCheckTest.State, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, lastCheckTest.SuppressedState, maintenanceInfo)
				require.NotNil(t, eventInfo)
				require.Equal(t, &moira.EventInfo{Maintenance: &maintenanceInfo}, eventInfo)
				require.True(t, needSend)
			})

			t.Run("Create EventInfo with interval", func(t *testing.T) {
				var interval int64 = 24

				eventInfo, needSend := triggerChecker.isStateChanged(moira.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp()-100000, lastCheckTest.Suppressed, moira.StateNODATA, moira.MaintenanceInfo{})
				require.NotNil(t, eventInfo)
				require.Equal(t, &moira.EventInfo{Interval: &interval}, eventInfo)
				require.True(t, needSend)
			})

			t.Run("No send message", func(t *testing.T) {
				eventInfo, needSend := triggerChecker.isStateChanged(moira.StateNODATA, lastCheckTest.State, currentCheckTest.Timestamp, lastCheckTest.GetEventTimestamp(), lastCheckTest.Suppressed, moira.StateNODATA, moira.MaintenanceInfo{})
				require.Nil(t, eventInfo)
				require.False(t, needSend)
			})
		})
	})
}
