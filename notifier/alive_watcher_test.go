package notifier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"go.uber.org/mock/gomock"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
)

func initAliveMeter(mockCtrl *gomock.Controller) (*mock_metrics.MockRegistry, *mock_metrics.MockMetricRegistry, *mock_metrics.MockMeter) {
	mockRegistry := mock_metrics.NewMockRegistry(mockCtrl)
	mockAliveMeter := mock_metrics.NewMockMeter(mockCtrl)
	mockAttributesRegistry := mock_metrics.NewMockMetricRegistry(mockCtrl)

	mockRegistry.EXPECT().NewMeter(gomock.Any()).Times(5)
	mockRegistry.EXPECT().NewHistogram(gomock.Any()).Times(3)
	mockRegistry.EXPECT().NewMeter("", "alive").Return(mockAliveMeter)

	mockAttributesRegistry.EXPECT().NewGauge(gomock.Any()).Times(5)
	mockAttributesRegistry.EXPECT().NewHistogram(gomock.Any()).Times(3)
	mockAttributesRegistry.EXPECT().NewGauge("alive").Return(mockAliveMeter)

	return mockRegistry, mockAttributesRegistry, mockAliveMeter
}

func TestAliveWatcher_checkNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	mockRegistry, mockAttributesRegistry, mockAliveMeter := initAliveMeter(mockCtrl)
	testNotifierMetrics := metrics.ConfigureNotifierMetrics(mockRegistry, mockAttributesRegistry, "")

	aliveWatcher := NewAliveWatcher(nil, dataBase, 0, testNotifierMetrics)

	t.Run("checkNotifierState", func(t *testing.T) {
		t.Run("when OK", func(t *testing.T) {
			dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
				State: moira.SelfStateOK,
				Actor: moira.SelfStateActorManual,
			}, nil)
			mockAliveMeter.EXPECT().Mark(int64(1)).Times(2)

			aliveWatcher.checkNotifierState()
		})

		t.Run("when not OK state and no errors", func(t *testing.T) {
			notOKStates := []string{moira.SelfStateERROR, "err", "bad", "", "1"}

			for _, badState := range notOKStates {
				dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
					State: badState,
					Actor: moira.SelfStateActorManual,
				}, nil)
				mockAliveMeter.EXPECT().Mark(int64(0)).Times(2)

				aliveWatcher.checkNotifierState()
			}
		})

		t.Run("when not OK state and errors", func(t *testing.T) {
			notOKState := ""
			givenErrors := []error{
				errors.New("one error"),
				errors.New("another error"),
			}

			for _, err := range givenErrors {
				dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
					State: notOKState,
				}, err)
				mockAliveMeter.EXPECT().Mark(int64(0)).Times(2)

				aliveWatcher.checkNotifierState()
			}
		})
	})
}

func TestAliveWatcher_Start(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := mock_moira_alert.NewMockLogger(mockCtrl)
	eventsBuilder := mock_moira_alert.NewMockEventBuilder(mockCtrl)
	logger.EXPECT().Info().Return(eventsBuilder).AnyTimes()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	const (
		testCheckNotifierStateTimeout = time.Second
	)

	mockRegistry, mockAttributesRegistry, mockAliveMeter := initAliveMeter(mockCtrl)
	testNotifierMetrics := metrics.ConfigureNotifierMetrics(mockRegistry, mockAttributesRegistry, "")

	aliveWatcher := NewAliveWatcher(logger, dataBase, testCheckNotifierStateTimeout, testNotifierMetrics)

	t.Run("AliveWatcher stops on cancel", func(t *testing.T) {
		eventsBuilder.EXPECT().
			Interface("check_timeout_seconds", testCheckNotifierStateTimeout.Seconds()).
			Return(eventsBuilder)
		eventsBuilder.EXPECT().Msg("Moira Notifier alive watcher started")
		eventsBuilder.EXPECT().Msg("Moira Notifier alive watcher stopped")

		dataBase.EXPECT().GetNotifierStateForSource(moira.DefaultLocalCluster).Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}, nil).AnyTimes()
		mockAliveMeter.EXPECT().Mark(int64(1)).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		aliveWatcher.Start(ctx)

		time.Sleep(time.Second * 3)
		cancel()
		time.Sleep(time.Millisecond)
	})
}
