package heartbeat

import (
	"errors"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGraphiteRemoteChecker(t *testing.T) {
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	Convey("Test remote checker heartbeat", t, func() {
		err := errors.New("test error remoteChecker")
		now := time.Now().Unix()

		check, mockCtrl := createGraphiteRemoteCheckerTest(t)
		defer mockCtrl.Finish()

		database := check.database.(*mock_moira_alert.MockDatabase)

		Convey("Checking the created graphite remote checker", func() {
			expected := &remoteChecker{heartbeat: heartbeat{database: check.database, logger: check.logger, delay: 1, lastSuccessfulCheck: now, checkTags: check.checkTags}}
			So(GetRemoteChecker(0, now, check.checkTags, check.logger, check.database), ShouldBeNil)
			So(GetRemoteChecker(1, now, check.checkTags, check.logger, check.database), ShouldResemble, expected)
		})

		Convey("GraphiteRemoteChecker error handling test", func() {
			database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(0), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldEqual, err)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test update lastSuccessfulCheck", func() {
			now += 1000

			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Check for notification", func() {
			check.lastSuccessfulCheck = now - check.delay - 1

			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, now-check.lastSuccessfulCheck)
		})

		Convey("Exit without action", func() {
			database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultRemoteCluster).Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test NeedToCheckOthers and NeedTurnOffNotifier", func() {
			// TODO(litleleprikon): seems that this test checks nothing. Seems that NeedToCheckOthers and NeedTurnOffNotifier do not work.
			So(check.NeedToCheckOthers(), ShouldBeTrue)
			So(check.NeedTurnOffNotifier(), ShouldBeFalse)
		})
	})
}

func createGraphiteRemoteCheckerTest(t *testing.T) (*remoteChecker, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("MetricDelay")
	checkTags := []string{}

	return GetRemoteChecker(120, time.Now().Unix(), checkTags, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*remoteChecker), mockCtrl
}
