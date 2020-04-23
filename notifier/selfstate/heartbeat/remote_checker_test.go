package heartbeat

import (
	"errors"
	"testing"
	"time"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGraphiteRemoteChecker(t *testing.T) {
	Convey("Test remote checker heartbeat", t, func() {
		err := errors.New("test error remoteChecker")
		now := time.Now().Unix()
		check := createGraphiteRemoteCheckerTest(t)
		database := check.database.(*mock_moira_alert.MockDatabase)

		Convey("Checking the created graphite remote checker", func() {
			expected := &remoteChecker{heartbeat: heartbeat{database: check.database, logger: check.logger, delay: 1, lastSuccessfulCheck: now}}
			So(GetRemoteChecker(0, check.logger, check.database), ShouldBeNil)
			So(GetRemoteChecker(1, check.logger, check.database), ShouldResemble, expected)
		})

		Convey("GraphiteRemoteChecker error handling test", func() {
			database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(0), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldEqual, err)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test update lastSuccessfulCheck", func() {
			now += 1000
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
			database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Check for notification", func() {
			check.lastSuccessfulCheck = now - check.delay - 1

			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, now-check.lastSuccessfulCheck)
		})

		Convey("Exit without action", func() {
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test NeedToCheckOthers and NeedTurnOffNotifier", func() {
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
			database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(0), nil)
			So(check.NeedToCheckOthers(), ShouldBeTrue)

			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(0), nil)
			So(check.NeedToCheckOthers(), ShouldBeTrue)

			So(check.NeedTurnOffNotifier(), ShouldBeFalse)
		})
	})
}

func createGraphiteRemoteCheckerTest(t *testing.T) *remoteChecker {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("MetricDelay")

	return GetRemoteChecker(120, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*remoteChecker)
}
