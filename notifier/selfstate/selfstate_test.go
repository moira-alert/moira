package selfstate

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/mock/notifier"
	"github.com/moira-alert/moira-alert/notifier"
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
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		nextSendErrorMessage int64
	)

	// _, selfStateWorker, database, notif, conf, mockCtrl := configureWorker(t)
	mock := configureWorker(t)
	mock.selfCheckWorker.Start()
	Convey("Database disconnected", t, func() {
		Convey("Should notify admin", func() {
			var sendingWG sync.WaitGroup
			err := fmt.Errorf("DataBase doesn't work")
			mock.database.EXPECT().GetMetricsCount().Return(int64(1), nil)
			mock.database.EXPECT().GetChecksCount().Return(int64(1), err)

			now := time.Now()
			redisLastCheckTS = now.Add(-time.Second * 11).Unix()
			lastCheckTS = now.Unix()
			nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
			lastMetricReceivedTS = now.Unix()
			expectedPackage := configureNotificationPackage(adminContact, mock.conf.RedisDisconnectDelay, now.Unix()-redisLastCheckTS, "Redis disconnected")

			mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
			mock.selfCheckWorker.check(now.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

			So(lastMetricReceivedTS, ShouldEqual, now.Unix())
			So(lastCheckTS, ShouldEqual, now.Unix())
			So(redisLastCheckTS, ShouldEqual, now.Add(-time.Second*11).Unix())
			So(nextSendErrorMessage, ShouldEqual, now.Unix()+mock.conf.NoticeInterval)
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
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		nextSendErrorMessage int64
	)

	mock := configureWorker(t)
	mock.selfCheckWorker.Start()
	Convey("Should notify admin", t, func() {
		var sendingWG sync.WaitGroup
		mock.database.EXPECT().GetMetricsCount().Return(int64(1), nil)
		mock.database.EXPECT().GetChecksCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Add(-time.Second * 61).Unix()
		metricsCount = 1

		callingNow := now.Add(time.Second * 2)
		expectedPackage := configureNotificationPackage(adminContact, mock.conf.LastMetricReceivedDelay, callingNow.Unix()-lastMetricReceivedTS, "Moira-Cache does not received new metrics")

		mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
		mock.selfCheckWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

		So(lastMetricReceivedTS, ShouldEqual, now.Add(-time.Second*61).Unix())
		So(lastCheckTS, ShouldEqual, callingNow.Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+mock.conf.NoticeInterval)
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
		lastMetricReceivedTS int64
		redisLastCheckTS     int64
		lastCheckTS          int64
		nextSendErrorMessage int64
	)

	mock := configureWorker(t)
	mock.selfCheckWorker.Start()
	Convey("Should notify admin", t, func() {
		var sendingWG sync.WaitGroup
		mock.database.EXPECT().GetMetricsCount().Return(int64(1), nil)
		mock.database.EXPECT().GetChecksCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Add(-time.Second * 121).Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Unix()
		checksCount = 1

		callingNow := now.Add(time.Second * 2)
		expectedPackage := configureNotificationPackage(adminContact, mock.conf.LastCheckDelay, callingNow.Unix()-lastCheckTS, "Moira-Checker does not checks triggers")

		mock.notif.EXPECT().Send(&expectedPackage, &sendingWG)
		mock.selfCheckWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

		So(lastMetricReceivedTS, ShouldEqual, callingNow.Unix())
		So(lastCheckTS, ShouldEqual, now.Add(-time.Second*121).Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+mock.conf.NoticeInterval)
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
		RedisDisconnectDelay:    10,
		LastMetricReceivedDelay: 60,
		LastCheckDelay:          120,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	senders := map[string]bool{
		"admin-mail": true,
	}
	notif.EXPECT().GetSenders().Return(senders)

	selfStateWorker := &SelfCheckWorker{
		Log:      logger,
		DB:       database,
		Config:   conf,
		Notifier: notif,
	}

	Convey("Go routine run before first send, should send after 10 seconds next time", t, func() {
		err := fmt.Errorf("DataBase doesn't work")
		database.EXPECT().GetMetricsCount().Return(int64(1), nil).Times(11)
		database.EXPECT().GetChecksCount().Return(int64(1), err).Times(11)
		notif.EXPECT().Send(gomock.Any(), gomock.Any())
		selfStateWorker.Start()
		time.Sleep(time.Second*11 + time.Millisecond*500)
		selfStateWorker.Stop()
	})
}

func configureWorker(t *testing.T) *selfCheckWorkerMock {
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
		RedisDisconnectDelay:    10,
		LastMetricReceivedDelay: 60,
		LastCheckDelay:          120,
		NoticeInterval:          60,
	}

	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif := mock_notifier.NewMockNotifier(mockCtrl)
	senders := map[string]bool{
		"admin-mail": true,
	}
	notif.EXPECT().GetSenders().Return(senders)

	return &selfCheckWorkerMock{
		selfCheckWorker: &SelfCheckWorker{
			Log:      logger,
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

func configureNotificationPackage(adminContact map[string]string, errorValue int64, currentValue int64, message string) notifier.NotificationPackage {
	val := float64(currentValue)
	return notifier.NotificationPackage{
		Contact: moira.ContactData{
			Type:  adminContact["type"],
			Value: adminContact["value"],
		},
		Trigger: moira.TriggerData{
			Name:       message,
			ErrorValue: float64(errorValue),
		},
		Events: []moira.EventData{
			{
				Timestamp: time.Now().Unix(),
				State:     "ERROR",
				Metric:    message,
				Value:     &val,
			},
		},
		DontResend: true,
	}
}
