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

func TestFilter(t *testing.T) {
	Convey("Test filter heartbeat", t, func() {
		err := errors.New("test filter error")
		now := time.Now().Unix()

		check, mockCtrl := createFilterTest(t)
		defer mockCtrl.Finish()

		database := check.database.(*mock_moira_alert.MockDatabase)
		defaultLocalCluster := moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster)

		Convey("Checking the created filter", func() {
			expected := &filter{
				heartbeat: heartbeat{
					database:            check.database,
					logger:              check.logger,
					delay:               1,
					lastSuccessfulCheck: now,
					checkTags:           check.checkTags,
				},
			}

			So(GetFilter(0, now, check.checkTags, check.logger, check.database), ShouldBeNil)
			So(GetFilter(1, now, check.checkTags, check.logger, check.database), ShouldResemble, expected)
		})

		Convey("Filter error handling test", func() {
			database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), err)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldEqual, err)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test update lastSuccessfulCheck", func() {
			now += 1000

			database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
			So(check.lastSuccessfulCheck, ShouldResemble, now)
		})

		Convey("Check for notification", func() {
			check.lastSuccessfulCheck = now - check.delay - 1

			database.EXPECT().GetMetricsUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, now-check.lastSuccessfulCheck)
		})

		Convey("Exit without action", func() {
			database.EXPECT().GetMetricsUpdatesCount().Return(int64(0), nil)
			database.EXPECT().GetTriggersToCheckCount(defaultLocalCluster).Return(int64(1), nil)

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

func createFilterTest(t *testing.T) (*filter, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("MetricDelay")
	checkTags := []string{}

	return GetFilter(60, time.Now().Unix(), checkTags, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*filter), mockCtrl
}
