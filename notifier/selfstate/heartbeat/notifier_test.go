package heartbeat

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNotifierState(t *testing.T) {
	t.Run("Test notifier delay heartbeat", func(t *testing.T) {
		now := time.Now().Unix()
		check := createNotifierStateTest(t)

		t.Run("Test get notifier delay", func(t *testing.T) {
			check.database.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				State: moira.SelfStateOK,
				Actor: moira.SelfStateActorManual,
			}, nil)

			value, needSend, errActual := check.Check(now)
			require.NoError(t, errActual)
			require.False(t, needSend)
			require.EqualValues(t, 0, value)
		})

		t.Run("Test get notification", func(t *testing.T) {
			check.database.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				State: moira.SelfStateERROR,
				Actor: moira.SelfStateActorManual,
			}, nil).Times(2)

			value, needSend, errActual := check.Check(now)
			require.NoError(t, errActual)
			require.True(t, needSend)
			require.EqualValues(t, 0, value)
		})

		t.Run("Should return OK if notifier disabled automatically", func(t *testing.T) {
			check.database.(*mock_moira_alert.MockDatabase).EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				State: moira.SelfStateERROR,
				Actor: moira.SelfStateActorAutomatic,
			}, nil)

			value, hasError, err := check.Check(now)
			require.NoError(t, err)
			require.False(t, hasError)
			require.EqualValues(t, 0, value)
		})

		t.Run("Test NeedToCheckOthers and NeedTurnOffNotifier", func(t *testing.T) {
			require.False(t, check.NeedTurnOffNotifier())
			require.True(t, check.NeedToCheckOthers())
		})
	})
}

func createNotifierStateTest(t *testing.T) *notifier {
	mockCtrl := gomock.NewController(t)
	logger, _ := logging.GetLogger("MetricDelay")
	checkTags := []string{}

	return GetNotifier(checkTags, "moira-system-disable-notification", []string{"moira-local-fatal"}, moira.DefaultLocalCluster, logger, mock_moira_alert.NewMockDatabase(mockCtrl)).(*notifier)
}
