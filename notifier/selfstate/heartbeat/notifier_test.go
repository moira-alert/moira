package heartbeat

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNotifierState(t *testing.T) {
	Convey("Test notifier delay heartbeat", t, func() {
		now := time.Now().Unix()
		check := createNotifierStateTest(t)

		Convey("Test get notifier delay", func() {
			check.db.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test get notification", func() {
			check.db.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil).Times(2)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeTrue)
			So(value, ShouldEqual, 0)
		})

		Convey("Test NeedToCheckOthers and NeedTurnOffNotifier", func() {
			So(check.NeedTurnOffNotifier(), ShouldBeFalse)
			So(check.NeedToCheckOthers(), ShouldBeTrue)
		})
	})
}

func createNotifierStateTest(t *testing.T) *notifier {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("MetricDelay")

	return GetNotifier(logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*notifier)
}
