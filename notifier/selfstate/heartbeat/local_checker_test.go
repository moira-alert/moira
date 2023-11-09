package heartbeat

import (
	"errors"
	"testing"
	"time"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCheckDelay_Check(t *testing.T) {
	Convey("Test local checker heartbeat", t, func() {
		err := errors.New("test error localChecker")
		now := time.Now().Unix()
		check, mockCtrl := createGraphiteLocalCheckerTest(t)
		defer mockCtrl.Finish()
		database := check.database.(*mock_moira_alert.MockDatabase)

		Convey("Test creation localChecker", func() {
			expected := &localChecker{heartbeat: heartbeat{database: check.database, logger: check.logger, delay: 1, lastSuccessfulCheck: now}}
			So(GetLocalChecker(0, check.logger, check.database), ShouldBeNil)
			So(GetLocalChecker(1, check.logger, check.database), ShouldResemble, expected)
		})

		Convey("GraphiteLocalChecker error handling test", func() {
			database.EXPECT().GetLocalTriggersToCheckCount().Return(int64(1), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldEqual, err)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test update lastSuccessfulCheck", func() {
			now += 1000
			database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil)
			database.EXPECT().GetLocalTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Test get notification", func() {
			check.lastSuccessfulCheck = now - check.delay - 1
			database.EXPECT().GetChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetLocalTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, now-check.lastSuccessfulCheck)
		})

		Convey("Exit without action", func() {
			database.EXPECT().GetChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetLocalTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test NeedToCheckOthers and NeedTurnOffNotifier", func() {
			// TODO(litleleprikon): seems that this test checks nothing. Seems that NeedToCheckOthers and NeedTurnOffNotifier do not work.
			needCheck := check.NeedToCheckOthers()
			So(needCheck, ShouldBeTrue)

			So(check.NeedTurnOffNotifier(), ShouldBeFalse)
		})
	})
}

func createGraphiteLocalCheckerTest(t *testing.T) (*localChecker, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("CheckDelay")

	return GetLocalChecker(120, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*localChecker), mockCtrl
}
