package checker

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCompareMetricStates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		TriggerID: "SuperId",
		Database:  dataBase,
		Logger:    logger,
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
		State:      NODATA,
	}

	Convey("Same state values", t, func() {
		Convey("Status OK, no need to send", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = OK
			currentState.State = OK

			actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, currentState)
		})

		Convey("Status NODATA and no remind interval, no need to send", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = NODATA
			currentState.State = NODATA

			actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, currentState)
		})

		Convey("Status ERROR and no remind interval, no need to send", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = ERROR
			currentState.State = ERROR

			actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, currentState)
		})

		Convey("Status NODATA and remind interval, need to send", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = NODATA
			currentState.State = NODATA
			currentState.Timestamp = 1502809200

			message := fmt.Sprintf("This metric has been in bad state for more than 24 hours - please, fix.")
			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				TriggerID: triggerChecker.TriggerID,
				Timestamp: currentState.Timestamp,
				State:     NODATA,
				OldState:  NODATA,
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
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = ERROR
			currentState.State = ERROR
			currentState.Timestamp = 1502809200

			message := fmt.Sprintf("This metric has been in bad state for more than 24 hours - please, fix.")
			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				TriggerID: triggerChecker.TriggerID,
				Timestamp: currentState.Timestamp,
				State:     ERROR,
				OldState:  ERROR,
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
			lastState.State = EXCEPTION
			currentState.State = EXCEPTION

			actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, currentState)
		})
	})

	Convey("Test different states", t, func() {
		Convey("Trigger maintenance", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = EXCEPTION
			currentState.State = OK
			currentState.SuppressedState = lastState.State
			currentState.Maintenance = 1502719222

			actual, err := triggerChecker.compareMetricStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = currentState.Timestamp
			currentState.Suppressed = true
			So(actual, ShouldResemble, currentState)
		})
	})
}

func TestCompareTriggerStates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		TriggerID: "SuperId",
		Database:  dataBase,
		Logger:    logger,
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
			lastCheck.State = OK
			currentCheck.State = OK
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
			lastCheck.State = OK
			currentCheck.State = NODATA
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
		Logger:   logger,
		Until:    67,
		From:     17,
		Database: dataBase,
		trigger: &moira.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
	}

	lastState := moira.MetricState{
		Timestamp:      1000,
		EventTimestamp: 1,
		Suppressed:     true,
		Maintenance:    1100,
		State:          WARN,
	}
	currentState := moira.MetricState{
		Suppressed: false,
		Timestamp:  1200,
		State:      WARN,
	}

	states := []string{OK, WARN, ERROR, NODATA, EXCEPTION, DEL}

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
		Logger:   logger,
		Until:    67,
		From:     17,
		Database: dataBase,
		trigger: &moira.Trigger{
			ID:      "superId",
			Targets: []string{"aliasByNode(super.*.metric, 0)"},
		},
		lastCheck: &moira.CheckData{},
	}

	lastState := moira.MetricState{
		Timestamp:      100,
		EventTimestamp: 10,
		State:          OK,
	}
	currentState := moira.MetricState{
		Timestamp:   1000,
		Maintenance: 1500,
		State:       WARN,
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
			currentState.State = OK
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test still in maintenance. State changes OK => ERROR", func() {
			lastState = currentState
			currentState.Timestamp = 1200
			currentState.State = ERROR
			actual, err := triggerChecker.compareMetricStates("super.awesome.metric", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = currentState.Timestamp
			So(actual, ShouldResemble, currentState)
		})

		Convey("Test switch out of maintenance. State didn't change", func() {
			lastState = currentState
			currentState.Timestamp = 1600
			currentState.State = OK
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
				SuppressedState: OK,
				State:           ERROR,
			}
			currentState.Timestamp = 1600
			currentState.State = ERROR

			message := fmt.Sprintf("This metric changed its state during maintenance interval.")
			dataBase.EXPECT().PushNotificationEvent(&moira.NotificationEvent{
				TriggerID: triggerChecker.TriggerID,
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
