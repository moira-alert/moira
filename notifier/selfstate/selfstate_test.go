package selfstate

import (
	"errors"
	"testing"
	"time"

	mock_heartbeat "github.com/moira-alert/moira/mock/heartbeat"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"

	"github.com/moira-alert/moira"

	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

type selfCheckWorkerMock struct {
	selfCheckWorker *SelfCheckWorker
	database        *mock_moira_alert.MockDatabase
	notif           *mock_notifier.MockNotifier
	conf            Config
	mockCtrl        *gomock.Controller
}

func TestSelfCheckWorker_selfStateChecker(t *testing.T) {
	mock := configureWorker(t, true)
	mock.selfCheckWorker.Start()
	Convey("Test creation all heartbeat", t, func() {
		var nextSendErrorMessage int64
		var events []moira.NotificationEvent
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil).Times(2)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
		mock.database.EXPECT().GetRemoteTriggersToCheckCount().Return(int64(1), nil)
		mock.database.EXPECT().GetLocalTriggersToCheckCount().Return(int64(1), nil).Times(2)
		mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

		mock.selfCheckWorker.sendErrorMessages(events)
		time.Sleep(time.Millisecond)
		mock.selfCheckWorker.check(time.Now().Unix(), nextSendErrorMessage)

		So(len(mock.selfCheckWorker.Heartbeats), ShouldEqual, 5)
	})

	mock.selfCheckWorker.Stop()
	mock.mockCtrl.Finish()
}

func TestSelfCheckWorker_Start(t *testing.T) {
	mock := configureWorker(t, false)

	Convey("Test start selfCheckWorkerMock", t, func() {
		Convey("Test enabled is false", func() {
			mock.selfCheckWorker.Config.Enabled = false
			mock.selfCheckWorker.Start()
			So(mock.selfCheckWorker.Heartbeats, ShouldBeNil)
		})
		Convey("Check for error from checkConfig", func() {
			mock.selfCheckWorker.Config.Enabled = true
			mock.notif.EXPECT().GetSenders().Return(nil)
			mock.selfCheckWorker.Start()
			So(mock.selfCheckWorker.Heartbeats, ShouldBeNil)
		})
	})
}

func TestSelfCheckWorker_Stop(t *testing.T) {
	Convey("Test stop selfCheckWorkerMock", t, func() {
		mock := configureWorker(t, false)
		Convey("Test enabled is false", func() {
			mock.selfCheckWorker.Config.Enabled = false
			So(mock.selfCheckWorker.Stop(), ShouldBeNil)
		})
		Convey("Check for error from checkConfig", func() {
			mock.selfCheckWorker.Config.Enabled = true
			mock.notif.EXPECT().GetSenders().Return(nil)
			So(mock.selfCheckWorker.Stop(), ShouldBeNil)
		})
	})
}

func TestSelfCheckWorker(t *testing.T) {
	Convey("Test checked heartbeat", t, func() {
		err := errors.New("test error")
		now := time.Now().Unix()

		mock := configureWorker(t, false)

		Convey("Test handle error and no needed send events", func() {
			check := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			mock.selfCheckWorker.Heartbeats = []heartbeat.Heartbeater{check}

			check.EXPECT().Check(now).Return(int64(0), false, err)

			events := mock.selfCheckWorker.handleCheckServices(now)
			So(events, ShouldBeNil)
		})

		Convey("Test turn off notification", func() {
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.Heartbeats = []heartbeat.Heartbeater{first, second}

			first.EXPECT().NeedTurnOffNotifier().Return(true)
			first.EXPECT().NeedToCheckOthers().Return(false)
			first.EXPECT().GetErrorMessage().Return(moira.SelfStateERROR)
			first.EXPECT().Check(now).Return(int64(0), true, nil)
			mock.database.EXPECT().SetNotifierState(moira.SelfStateERROR)

			events := mock.selfCheckWorker.handleCheckServices(now)
			So(len(events), ShouldEqual, 1)
		})

		Convey("Test of sending notifications from a check", func() {
			now = time.Now().Unix()
			first := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)
			second := mock_heartbeat.NewMockHeartbeater(mock.mockCtrl)

			mock.selfCheckWorker.Heartbeats = []heartbeat.Heartbeater{first, second}
			nextSendErrorMessage := time.Now().Unix() - time.Hour.Milliseconds()

			first.EXPECT().Check(now).Return(int64(0), true, nil)
			first.EXPECT().GetErrorMessage().Return(moira.SelfStateERROR)
			first.EXPECT().NeedTurnOffNotifier().Return(true)
			first.EXPECT().NeedToCheckOthers().Return(false)
			mock.database.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(err)
			mock.notif.EXPECT().Send(gomock.Any(), gomock.Any())

			nextSendErrorMessage = mock.selfCheckWorker.check(now, nextSendErrorMessage)
			So(nextSendErrorMessage, ShouldEqual, now+60)
		})

		mock.mockCtrl.Finish()
	})
}

func configureWorker(t *testing.T, isStart bool) *selfCheckWorkerMock {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}
	defaultCheckInterval = time.Second * 1
	conf := Config{
		Enabled: true,
		Contacts: []map[string]string{
			adminContact,
		},
		RedisDisconnectDelaySeconds:    10,
		LastMetricReceivedDelaySeconds: 60,
		LastCheckDelaySeconds:          120,
		NoticeIntervalSeconds:          60,
		LastRemoteCheckDelaySeconds:    120,
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	if isStart {
		senders := map[string]bool{
			"admin-mail": true,
		}
		notif.EXPECT().GetSenders().Return(senders).MinTimes(1)

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil)
		lock.EXPECT().Release()
		database.EXPECT().NewLock(gomock.Any(), gomock.Any()).Return(lock)
	}

	return &selfCheckWorkerMock{
		selfCheckWorker: &SelfCheckWorker{
			Logger:   logger,
			Database: database,
			Config:   conf,
			Notifier: notif,
		},
		database: database,
		notif:    notif,
		conf:     conf,
		mockCtrl: mockCtrl,
	}
}
