package heartbeat

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func heartbeaterHelper(t *testing.T) (*mock_moira_alert.MockDatabase, *mock_clock.MockClock, time.Time, *heartbeaterBase) {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger, _ := logging.GetLogger("Test")
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	clock := mock_clock.NewMockClock(mockCtrl)

	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	clock.EXPECT().NowUTC().Return(testTime)
	heartbeaterBase := NewHeartbeaterBase(logger, database, clock)

	return database, clock, testTime, heartbeaterBase
}

func TestStateIsDegradated(t *testing.T) {
	Convey("Test state.IsDegradated", t, func() {
		Convey("With degradated state", func() {
			lastState := StateOK
			newState := StateError

			degradated := lastState.IsDegraded(newState)
			So(degradated, ShouldBeTrue)
		})

		Convey("Without degradated state", func() {
			lastState := StateError
			newState := StateOK

			degradated := lastState.IsDegraded(newState)
			So(degradated, ShouldBeFalse)
		})
	})
}

func TestStateIsRecovered(t *testing.T) {
	Convey("Test state.IsRecovered", t, func() {
		Convey("With recovered state", func() {
			lastState := StateError
			newState := StateOK

			recovered := lastState.IsRecovered(newState)
			So(recovered, ShouldBeTrue)
		})

		Convey("Without recovered state", func() {
			lastState := StateOK
			newState := StateError

			recovered := lastState.IsRecovered(newState)
			So(recovered, ShouldBeFalse)
		})
	})
}

func TestNewHeartbeaterBase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger, _ := logging.GetLogger("Test")
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	clock := mock_clock.NewMockClock(mockCtrl)

	testTime := time.Date(2022, time.June, 6, 10, 0, 0, 0, time.UTC)

	Convey("Test NewHeartbeaterBase", t, func() {
		clock.EXPECT().NowUTC().Return(testTime)

		expected := &heartbeaterBase{
			logger:   logger,
			database: database,
			clock:    clock,

			lastSuccessfulCheck: testTime,
		}

		heartbeaterBase := NewHeartbeaterBase(logger, database, clock)
		So(heartbeaterBase, ShouldResemble, expected)
	})
}

func TestValidateHeartbeaterBaseConfig(t *testing.T) {
	Convey("Test validation heartbeaterBaseConfig", t, func() {
		Convey("With disabled config", func() {
			hbCfg := HeartbeaterBaseConfig{}
			err := moira.ValidateStruct(hbCfg)
			So(err, ShouldBeNil)
		})

		Convey("With enabled config, added and filled alert config", func() {
			hbCfg := HeartbeaterBaseConfig{
				Enabled: true,
				AlertCfg: AlertConfig{
					Name: "test name",
				},
			}
			err := moira.ValidateStruct(hbCfg)
			So(err, ShouldBeNil)
		})
	})
}
