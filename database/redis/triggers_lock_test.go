package redis

import (
	"testing"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLock(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	Convey("Test lock manipulation", t, func() {
		triggerID1 := "id"

		isSet, err := dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeTrue)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeFalse)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		So(err, ShouldNotBeNil)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)

		err = dataBase.AcquireTriggerCheckLock(triggerID1, 1)
		So(err, ShouldBeNil)

		isSet, err = dataBase.SetTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
		So(isSet, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock(triggerID1)
		So(err, ShouldBeNil)
	})
}

func TestLockErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	Convey("Should throw error when no connection", t, func() {
		err := dataBase.AcquireTriggerCheckLock("tr1", 4)
		So(err, ShouldNotBeNil)

		actual, err := dataBase.SetTriggerCheckLock("tr1")
		So(err, ShouldNotBeNil)
		So(actual, ShouldBeFalse)

		err = dataBase.DeleteTriggerCheckLock("tr1")
		So(err, ShouldNotBeNil)
	})
}

func TestLockErrorLogging(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	logger := mock_moira_alert.NewMockLogger(mockCtrl)
	eventBuilder := mock_moira_alert.NewMockEventBuilder(mockCtrl)
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)

	dataBase.Flush()
	defer dataBase.Flush()
	Convey("Should log error on releasing the lock", t, func() {
		logger.EXPECT().Warningb().Return(eventBuilder).AnyTimes()
		eventBuilder.EXPECT().Error(gomock.Any()).Return(eventBuilder)
		eventBuilder.EXPECT().Msg(gomock.Any())

		dataBase.ReleaseTriggerCheckLock("tr1")
	})
}
