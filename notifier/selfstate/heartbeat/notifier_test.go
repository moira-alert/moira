package heartbeat

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestNotifierState(t *testing.T) {
	Convey("Test notifier delay heartbeat", t, func() {
		now := time.Now().Unix()
		check := createNotifierStateTest(t)

		Convey("Test get notifier delay", func() {
			check.database.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierState().Return(moira.NotifierState{
				OldState: moira.SelfStateOK,
				NewState: moira.SelfStateOK,
				Actor:    moira.SelfStateActorManual,
			}, nil)

			value, needSend, errActual := check.Check(now)
			So(errActual, ShouldBeNil)
			So(needSend, ShouldBeFalse)
			So(value, ShouldEqual, 0)
		})

		Convey("Test get notification", func() {
			check.database.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierState().Return(moira.NotifierState{
				OldState: moira.SelfStateERROR,
				NewState: moira.SelfStateERROR,
				Actor:    moira.SelfStateActorManual,
			}, nil).Times(2)

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
	checkTags := []string{}

	return GetNotifier(checkTags, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*notifier)
}
