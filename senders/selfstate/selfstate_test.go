package selfstate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
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

	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := Sender{Database: dataBase, logger: logger}

	t.Run("Has connection to database", func(t *testing.T) {
		t.Run("SelfState is OK", func(t *testing.T) {
			selfStateInitial := moira.SelfStateOK
			selfStateFinal := moira.SelfStateERROR

			t.Run("Should ignore events received", func(t *testing.T) {
				for _, subjectState := range ignorableSubjectStates {
					testEvents := []moira.NotificationEvent{{State: subjectState}}
					dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
						State: selfStateInitial,
					}, nil)

					err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
					require.NoError(t, err)
				}
			})

			t.Run("Should disable notifications", func(t *testing.T) {
				for _, subjectState := range disablingSubjectStates {
					dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
						State: selfStateInitial,
					}, nil)
					dataBase.EXPECT().SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorTrigger, selfStateFinal).Return(nil)

					testEvents := []moira.NotificationEvent{{State: subjectState}}
					err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
					require.NoError(t, err)
				}
			})
		})

		t.Run("SelfState is ERROR", func(t *testing.T) {
			selfStateInitial := moira.SelfStateERROR

			for _, subjectState := range disablingSubjectStates {
				testEvents := []moira.NotificationEvent{{State: subjectState}}
				dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
					State: selfStateInitial,
				}, nil)

				err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
				require.NoError(t, err)
			}
		})
	})

	t.Run("Has no connections to database", func(t *testing.T) {
		sender := Sender{Database: dataBase, logger: logger}

		for _, subjectState := range disablingSubjectStates {
			testEvents := []moira.NotificationEvent{{State: subjectState}}

			dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{}, fmt.Errorf("redis is down"))

			err := sender.SendEvents(testEvents, testContact, testTrigger, testPlots, testThrottled)
			require.Error(t, err)
			require.Error(t, err, "failed to get notifier state: redis is down")
		}
	})
}
