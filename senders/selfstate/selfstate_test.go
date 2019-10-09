package selfstate

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
)

var (
	testTrigger = moira.TriggerData{
		ID: "testTriggerToDisableNotifications",
		Targets: []string{
			"aliasByNode(scaleToSeconds(nonNegativeDerivative(DevOps.moira.{moira-docker1,moira-docker2,moira-docker3}.filter.received.matching.count), 1), 2, 5)",
		},
		WarnValue:  float64(2000),
		ErrorValue: float64(1900),
		Desc:       "Too few matched metrics found",
	}
	testContact = moira.ContactData{
		Type: "selfstate",
	}
	testThrottled = false
	testPlots     = make([][]byte, 0)
)

var (
	ignorableSubjectStates = []moira.State{moira.StateTEST, moira.StateOK, moira.StateEXCEPTION}
	disablingSubjectStates = []moira.State{moira.StateWARN, moira.StateERROR, moira.StateNODATA}
)

func TestSender_SendEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	sender := Sender{Database: dataBase, logger: logger}

	Convey("Has connection to database", t, func() {

		Convey("SelfState is OK", func() {
			selfStateInitial := moira.SelfStateOK
			selfStateFinal := moira.SelfStateERROR

			Convey("Should ignore events received", func() {
				for _, subjectState := range ignorableSubjectStates {
					testEvents := []moira.NotificationEvent{{State: subjectState}}
					dataBase.EXPECT().GetNotifierState().Return(selfStateInitial, nil)
					err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
					So(err, ShouldBeNil)
				}
			})

			Convey("Should disable notifications", func() {

				for _, subjectState := range disablingSubjectStates {
					dataBase.EXPECT().GetNotifierState().Return(selfStateInitial, nil)
					dataBase.EXPECT().SetNotifierState(selfStateFinal).Return(nil)
					testEvents := []moira.NotificationEvent{{State: subjectState}}
					err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
					So(err, ShouldBeNil)
				}
			})
		})

		Convey("SelfState is ERROR", func() {
			selfStateInitial := moira.SelfStateERROR

			for _, subjectState := range disablingSubjectStates {
				testEvents := []moira.NotificationEvent{{State: subjectState}}
				dataBase.EXPECT().GetNotifierState().Return(selfStateInitial, nil)
				err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
				So(err, ShouldBeNil)
			}
		})
	})

	Convey("Has no connections to database", t, func() {
		sender := Sender{Database: dataBase, logger: logger}

		for _, subjectState := range disablingSubjectStates {
			testEvents := []moira.NotificationEvent{{State: subjectState}}
			dataBase.EXPECT().GetNotifierState().Return("", fmt.Errorf("redis is down"))
			err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "failed to get notifier state: redis is down")
		}
	})
}
