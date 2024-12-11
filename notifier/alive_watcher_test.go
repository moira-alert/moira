package notifier

import (
	"errors"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_metrics "github.com/moira-alert/moira/mock/moira-alert/metrics"
)

func initAliveMeter(mockCtrl *gomock.Controller) (*mock_metrics.MockRegistry, *mock_metrics.MockMeter) {
	mockRegistry := mock_metrics.NewMockRegistry(mockCtrl)
	mockAliveMeter := mock_metrics.NewMockMeter(mockCtrl)

	mockRegistry.EXPECT().NewMeter(gomock.Any()).Times(5)
	mockRegistry.EXPECT().NewHistogram(gomock.Any()).Times(3)
	mockRegistry.EXPECT().NewMeter("", "alive").Return(mockAliveMeter)

	return mockRegistry, mockAliveMeter
}

func TestAliveWatcher_checkNotifierState(t *testing.T) {
	logger, _ = logging.GetLogger("test alive watcher")

	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	testConf := Config{
		CheckNotifierStateTimeout: time.Second * 10,
	}

	mockRegistry, mockAliveMeter := initAliveMeter(mockCtrl)
	testNotifierMetrics := metrics.ConfigureNotifierMetrics(mockRegistry, "")

	aliveWatcher := NewAliveWatcher(logger, dataBase, testConf, testNotifierMetrics)

	Convey("checkNotifierState", t, func() {
		Convey("when OK", func() {
			dataBase.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
			mockAliveMeter.EXPECT().Mark(int64(1))

			aliveWatcher.checkNotifierState()
		})

		Convey("when not OK state and no errors", func() {
			notOKStates := []string{moira.SelfStateERROR, "err", "bad", "", "1"}

			for _, badState := range notOKStates {
				dataBase.EXPECT().GetNotifierState().Return(badState, nil)
				mockAliveMeter.EXPECT().Mark(int64(0))

				aliveWatcher.checkNotifierState()
			}
		})

		Convey("when not OK state and errors", func() {
			notOKState := ""
			givenErrors := []error{
				errors.New("one error"),
				errors.New("another error"),
			}

			for _, err := range givenErrors {
				dataBase.EXPECT().GetNotifierState().Return(notOKState, err)
				mockAliveMeter.EXPECT().Mark(int64(0))

				aliveWatcher.checkNotifierState()
			}
		})
	})
}
