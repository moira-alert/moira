package checker

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCompareStates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		TriggerId: "SuperId",
		Database:  dataBase,
		Logger:    logger,
		trigger:   &moira.Trigger{},
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

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
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

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
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

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
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
			dataBase.EXPECT().PushEvent(&moira.EventData{
				TriggerID: triggerChecker.TriggerId,
				Timestamp: currentState.Timestamp,
				State:     NODATA,
				OldState:  NODATA,
				Metric:    "m1",
				Value:     currentState.Value,
				Message:   &message,
			}, false).Return(nil)
			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
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
			dataBase.EXPECT().PushEvent(&moira.EventData{
				TriggerID: triggerChecker.TriggerId,
				Timestamp: currentState.Timestamp,
				State:     ERROR,
				OldState:  ERROR,
				Metric:    "m1",
				Value:     currentState.Value,
				Message:   &message,
			}, false).Return(nil)
			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
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

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = lastState.EventTimestamp
			currentState.Suppressed = false
			So(actual, ShouldResemble, currentState)
		})

		Convey("Status EXCEPTION and lastState.Suppressed=true", func() {
			lastState := lastStateExample
			currentState := currentStateExample
			lastState.State = EXCEPTION
			lastState.Suppressed = true
			currentState.State = EXCEPTION

			dataBase.EXPECT().PushEvent(&moira.EventData{
				TriggerID: triggerChecker.TriggerId,
				Timestamp: currentState.Timestamp,
				State:     EXCEPTION,
				OldState:  EXCEPTION,
				Metric:    "m1",
				Value:     currentState.Value,
				Message:   nil,
			}, false).Return(nil)

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = currentState.Timestamp
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
			currentState.Maintenance = 1502719222

			actual, err := triggerChecker.compareStates("m1", currentState, lastState)
			So(err, ShouldBeNil)
			currentState.EventTimestamp = currentState.Timestamp
			currentState.Suppressed = true
			So(actual, ShouldResemble, currentState)
		})
	})
}
func TestCompareChecks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	logger, _ := logging.GetLogger("Test")
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	triggerChecker := TriggerChecker{
		TriggerId: "SuperId",
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
			actual, err := triggerChecker.compareChecks(currentCheck)

			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = currentCheck.Timestamp
			So(actual, ShouldResemble, currentCheck)
		})

		Convey("Need send", func() {
			lastCheck := lastCheckExample
			currentCheck := currentCheckExample
			triggerChecker.lastCheck = &lastCheck
			lastCheck.State = EXCEPTION
			lastCheck.Suppressed = true
			currentCheck.State = EXCEPTION

			dataBase.EXPECT().PushEvent(&moira.EventData{
				TriggerID: triggerChecker.TriggerId,
				Timestamp: currentCheck.Timestamp,
				State:     EXCEPTION,
				OldState:  EXCEPTION,
				Metric:    "",
				Value:     nil,
				Message:   nil,
			}, false).Return(nil)

			actual, err := triggerChecker.compareChecks(currentCheck)
			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = currentCheck.Timestamp
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
			actual, err := triggerChecker.compareChecks(currentCheck)

			So(err, ShouldBeNil)
			currentCheck.EventTimestamp = currentCheck.Timestamp
			currentCheck.Suppressed = true
			So(actual, ShouldResemble, currentCheck)
		})
	})
}
