package selfstate

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	"github.com/moira-alert/moira/notifier"
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

func TestDatabaseDisconnected(t *testing.T) {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}

	var (
		metricsCount         int64
		checksCount          int64
		remoteChecksCount    int64
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		lastRemoteCheckTS    int64
		nextSendErrorMessage int64
	)

	// _, selfStateWorker, database, notif, conf, mockCtrl := configureWorker(t)
	mock := configureWorker(t, false)
	mock.selfCheckWorker.Start()
	Convey("Database disconnected", t, func() {
		Convey("Should notify admin", func() {
			var events []moira.NotificationEvent
			var sendingWG sync.WaitGroup
			err := fmt.Errorf("DataBase doesn't work")
			mock.database.EXPECT().GetPatterns().Return([]string{"p1"}, nil)
			mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
			mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), err)
			mock.database.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, err)

			now := time.Now()
			redisLastCheckTS = now.Add(-time.Second * 11).Unix()
			lastCheckTS = now.Unix()
			lastRemoteCheckTS = now.Unix()
			nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
			lastMetricReceivedTS = now.Unix()
			appendNotificationEvents(&events, redisDisconnectedErrorMessage, now.Unix()-redisLastCheckTS)
			appendNotificationEvents(&events, notifierStateErrorMessage(moira.SelfStateERROR), 0)
			expectedPackage := configureNotificationPackage(adminContact, &events)

			mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
			mock.selfCheckWorker.check(now.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &lastRemoteCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount, &remoteChecksCount)

			So(lastMetricReceivedTS, ShouldEqual, now.Unix())
			So(lastCheckTS, ShouldEqual, now.Unix())
			So(redisLastCheckTS, ShouldEqual, now.Add(-time.Second*11).Unix())
			So(nextSendErrorMessage, ShouldEqual, now.Unix()+mock.conf.NoticeIntervalSeconds)
		})
	})
	mock.selfCheckWorker.Stop()
	mock.mockCtrl.Finish()
}

func TestMoiraCacheDoesNotReceivedNewMetrics(t *testing.T) {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}

	var (
		metricsCount         int64
		checksCount          int64
		remoteChecksCount    int64
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		lastRemoteCheckTS    int64
		nextSendErrorMessage int64
	)

	mock := configureWorker(t, false)
	mock.selfCheckWorker.Start()
	Convey("Should notify admin", t, func() {
		var events []moira.NotificationEvent
		var sendingWG sync.WaitGroup
		mock.database.EXPECT().GetPatterns().Return([]string{"p1"}, nil)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Unix()
		lastRemoteCheckTS = now.Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Add(-time.Second * 61).Unix()
		metricsCount = 1

		callingNow := now.Add(time.Second * 2)
		appendNotificationEvents(&events, filterStateErrorMessage, callingNow.Unix()-lastMetricReceivedTS)
		appendNotificationEvents(&events, notifierStateErrorMessage(moira.SelfStateERROR), 0)
		expectedPackage := configureNotificationPackage(adminContact, &events)

		mock.database.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)
		mock.database.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)
		mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
		mock.selfCheckWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &lastRemoteCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount, &remoteChecksCount)

		So(lastMetricReceivedTS, ShouldEqual, now.Add(-time.Second*61).Unix())
		So(lastCheckTS, ShouldEqual, callingNow.Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+mock.conf.NoticeIntervalSeconds)
	})
	mock.selfCheckWorker.Stop()
	mock.mockCtrl.Finish()
}

func TestMoiraCheckerDoesNotChecksTriggers(t *testing.T) {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}

	var (
		metricsCount         int64
		checksCount          int64
		remoteChecksCount    int64
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		lastRemoteCheckTS    int64
		nextSendErrorMessage int64
	)

	mock := configureWorker(t, false)
	mock.selfCheckWorker.Start()
	Convey("Should notify admin", t, func() {
		var events []moira.NotificationEvent
		var sendingWG sync.WaitGroup
		mock.database.EXPECT().GetPatterns().Return([]string{"p1"}, nil)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Add(-time.Second * 121).Unix()
		lastRemoteCheckTS = now.Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Unix()
		checksCount = 1

		callingNow := now.Add(time.Second * 2)
		appendNotificationEvents(&events, checkerStateErrorMessage, callingNow.Unix()-lastCheckTS)
		appendNotificationEvents(&events, notifierStateErrorMessage(moira.SelfStateERROR), 0)
		expectedPackage := configureNotificationPackage(adminContact, &events)

		mock.database.EXPECT().SetNotifierState(moira.SelfStateERROR).Return(nil)
		mock.database.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, nil)
		mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
		mock.selfCheckWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &lastRemoteCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount, &remoteChecksCount)

		So(lastMetricReceivedTS, ShouldEqual, callingNow.Unix())
		So(lastCheckTS, ShouldEqual, now.Add(-time.Second*121).Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+mock.conf.NoticeIntervalSeconds)
	})
	mock.selfCheckWorker.Stop()
	mock.mockCtrl.Finish()
}

func TestMoiraCheckerDoesNotChecksRemoteTriggers(t *testing.T) {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}

	var (
		metricsCount         int64
		checksCount          int64
		remoteChecksCount    int64
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		lastRemoteCheckTS    int64
		nextSendErrorMessage int64
	)

	mock := configureWorker(t, true)
	mock.selfCheckWorker.Start()
	Convey("Should notify admin", t, func() {
		var events []moira.NotificationEvent
		var sendingWG sync.WaitGroup
		mock.database.EXPECT().GetPatterns().Return([]string{"p1"}, nil)
		mock.database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetChecksUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetRemoteChecksUpdatesCount().Return(int64(1), nil)
		mock.database.EXPECT().GetRemoteTriggerIDs().Return([]string{"t1"}, nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Unix()
		lastRemoteCheckTS = now.Add(-time.Second * 121).Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Unix()
		checksCount = 1
		remoteChecksCount = 1

		callingNow := now.Add(time.Second * 2)
		appendNotificationEvents(&events, remoteCheckerStateErrorMessage, callingNow.Unix()-lastRemoteCheckTS)
		expectedPackage := configureNotificationPackage(adminContact, &events)

		mock.database.EXPECT().GetNotifierState().Return(moira.SelfStateOK, nil)
		mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
		mock.selfCheckWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &lastRemoteCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount, &remoteChecksCount)

		So(lastMetricReceivedTS, ShouldEqual, callingNow.Unix())
		So(lastRemoteCheckTS, ShouldEqual, now.Add(-time.Second*121).Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+mock.conf.NoticeIntervalSeconds)
	})
	mock.selfCheckWorker.Stop()
	mock.mockCtrl.Finish()
}

func TestRunGoRoutine(t *testing.T) {
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
		RedisDisconnectDelaySeconds:    5,
		LastMetricReceivedDelaySeconds: 60,
		LastCheckDelaySeconds:          120,
		NoticeIntervalSeconds:          3,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	senders := map[string]bool{
		"admin-mail": true,
	}
	notif.EXPECT().GetSenders().Return(senders).MinTimes(1)
	lock := mock_moira_alert.NewMockLock(mockCtrl)
	lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil)
	lock.EXPECT().Release()
	database.EXPECT().NewLock(gomock.Any(), gomock.Any()).Return(lock)

	selfStateWorker := &SelfCheckWorker{
		Logger:   logger,
		DB:       database,
		Config:   conf,
		Notifier: notif,
	}

	Convey("Go routine run before first send, should send after 10 seconds next time", t, func() {
		err := fmt.Errorf("DataBase doesn't work")
		database.EXPECT().GetPatterns().Return([]string{"p1"}, nil).Times(11)
		database.EXPECT().GetMetricsUpdatesCount().Return(int64(1), nil).Times(11)
		database.EXPECT().GetChecksUpdatesCount().Return(int64(1), err).Times(11)
		database.EXPECT().GetNotifierState().Return(moira.SelfStateERROR, err).Times(3)
		notif.EXPECT().Send(gomock.Any(), gomock.Any()).Times(3)
		selfStateWorker.Start()
		time.Sleep(time.Second*11 + time.Millisecond*500)
		selfStateWorker.Stop()
	})
}

func configureWorker(t *testing.T, remoteEnabled bool) *selfCheckWorkerMock {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}
	defaultCheckInterval = time.Second * 1
	conf := Config{
		Enabled:               true,
		RemoteTriggersEnabled: remoteEnabled,
		Contacts: []map[string]string{
			adminContact,
		},
		RedisDisconnectDelaySeconds:    10,
		LastMetricReceivedDelaySeconds: 60,
		LastCheckDelaySeconds:          120,
		LastRemoteCheckDelaySeconds:    120,
		NoticeIntervalSeconds:          60,
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	senders := map[string]bool{
		"admin-mail": true,
	}
	notif.EXPECT().GetSenders().Return(senders).MinTimes(1)

	lock := mock_moira_alert.NewMockLock(mockCtrl)
	lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil)
	lock.EXPECT().Release()
	database.EXPECT().NewLock(gomock.Any(), gomock.Any()).Return(lock)

	return &selfCheckWorkerMock{
		selfCheckWorker: &SelfCheckWorker{
			Logger:   logger,
			DB:       database,
			Config:   conf,
			Notifier: notif,
		},
		database: database,
		notif:    notif,
		conf:     conf,
		mockCtrl: mockCtrl,
	}
}

func configureNotificationPackage(adminContact map[string]string, events *[]moira.NotificationEvent) notifier.NotificationPackage {
	return notifier.NotificationPackage{
		Contact: moira.ContactData{
			Type:  adminContact["type"],
			Value: adminContact["value"],
		},
		Trigger: moira.TriggerData{
			Name:       "Moira health check",
			ErrorValue: float64(0),
		},
		Events:     *events,
		DontResend: true,
	}
}
