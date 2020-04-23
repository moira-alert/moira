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

func TestDatabaseHeartbeat(t *testing.T) {
	Convey("Test database heartbeat", t, func() {
		now := time.Now().Unix()
		err := errors.New("test database error")
		check := createRedisDelayTest(t)
		database := check.database.(*mock_moira_alert.MockDatabase)

		Convey("Checking the created heartbeat database", func() {
			expected := &databaseHeartbeat{heartbeat{database: check.database, logger: check.logger, delay: 1, lastSuccessfulCheck: now}}

			So(GetDatabase(0, check.logger, check.database), ShouldBeNil)
			So(GetDatabase(1, check.logger, check.database), ShouldResemble, expected)
		})

		Convey("Test update lastSuccessfulCheck", func() {
			now += 1000
			database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Database error handling test", func() {
			database.EXPECT().GetChecksUpdatesCount().Return(int64(1), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Check for notification", func() {
			check.lastSuccessfulCheck = now - check.delay - 1

			database.EXPECT().GetChecksUpdatesCount().Return(int64(0), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, now-check.lastSuccessfulCheck)
		})

		Convey("Test NeedToCheckOthers and NeedTurnOffNotifier", func() {
			So(check.NeedTurnOffNotifier(), ShouldBeTrue)
			So(check.NeedToCheckOthers(), ShouldBeFalse)
		})
	})
}

func createRedisDelayTest(t *testing.T) *databaseHeartbeat {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("CheckDelay")

	return GetDatabase(10, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*databaseHeartbeat)
}
