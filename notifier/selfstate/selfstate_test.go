package selfstate

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/mock/notifier"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

func TestSelfStateInit(t *testing.T) {
	needRun, selfStateWorker, _, _, _, mockCtrl := configureWorker(t)
	defer mockCtrl.Finish()

	Convey("Check need run self state checker", t, func() {
		So(needRun, ShouldBeTrue)
		So(selfStateWorker, ShouldNotBeNil)
	})
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

	_, selfStateWorker, database, notif, conf, mockCtrl := configureWorker(t)
	defer mockCtrl.Finish()

	Convey("Database disconnected", t, func() {
		Convey("Should notify admin", func() {
			var sendingWG sync.WaitGroup
			err := fmt.Errorf("DataBase doesn't work")
			database.EXPECT().GetMetricsCount().Return(int64(1), nil)
			database.EXPECT().GetChecksCount().Return(int64(1), err)

			now := time.Now()
			redisLastCheckTS = now.Add(-time.Second * 11).Unix()
			lastCheckTS = now.Unix()
			nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
			lastMetricReceivedTS = now.Unix()
			expectedPackage := configureNotificationPackage(adminContact, conf.RedisDisconnectDelay, now.Unix()-redisLastCheckTS, "Redis disconnected")

			notif.EXPECT().Send(&expectedPackage, &sendingWG)
			selfStateWorker.check(now.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

			So(lastMetricReceivedTS, ShouldEqual, now.Unix())
			So(lastCheckTS, ShouldEqual, now.Unix())
			So(redisLastCheckTS, ShouldEqual, now.Add(-time.Second*11).Unix())
			So(nextSendErrorMessage, ShouldEqual, now.Unix()+conf.NoticeInterval)
			mockCtrl.Finish()
		})
	})
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

	_, selfStateWorker, database, notif, conf, mockCtrl := configureWorker(t)
	defer mockCtrl.Finish()

	Convey("Should notify admin", t, func() {
		var sendingWG sync.WaitGroup
		database.EXPECT().GetMetricsCount().Return(int64(1), nil)
		database.EXPECT().GetChecksCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Add(-time.Second * 61).Unix()
		metricsCount = 1

		callingNow := now.Add(time.Second * 2)
		expectedPackage := configureNotificationPackage(adminContact, conf.LastMetricReceivedDelay, callingNow.Unix()-lastMetricReceivedTS, "Moira-Cache does not received new metrics")

		notif.EXPECT().Send(&expectedPackage, &sendingWG)
		selfStateWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

		So(lastMetricReceivedTS, ShouldEqual, now.Add(-time.Second*61).Unix())
		So(lastCheckTS, ShouldEqual, callingNow.Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+conf.NoticeInterval)
		mockCtrl.Finish()
	})
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

	_, selfStateWorker, database, notif, conf, mockCtrl := configureWorker(t)
	defer mockCtrl.Finish()

	Convey("Should notify admin", t, func() {
		var sendingWG sync.WaitGroup
		database.EXPECT().GetMetricsCount().Return(int64(1), nil)
		database.EXPECT().GetChecksCount().Return(int64(1), nil)

		now := time.Now()
		redisLastCheckTS = now.Unix()
		lastCheckTS = now.Add(-time.Second * 121).Unix()
		nextSendErrorMessage = now.Add(-time.Second * 5).Unix()
		lastMetricReceivedTS = now.Unix()
		checksCount = 1

		callingNow := now.Add(time.Second * 2)
		expectedPackage := configureNotificationPackage(adminContact, conf.LastCheckDelay, callingNow.Unix()-lastCheckTS, "Moira-Checker does not checks triggers")

		notif.EXPECT().Send(&expectedPackage, &sendingWG)
		selfStateWorker.check(callingNow.Unix(), &lastMetricReceivedTS, &redisLastCheckTS, &lastCheckTS, &nextSendErrorMessage, &metricsCount, &checksCount)

		So(lastMetricReceivedTS, ShouldEqual, callingNow.Unix())
		So(lastCheckTS, ShouldEqual, now.Add(-time.Second*121).Unix())
		So(redisLastCheckTS, ShouldEqual, callingNow.Unix())
		So(nextSendErrorMessage, ShouldEqual, callingNow.Unix()+conf.NoticeInterval)
		mockCtrl.Finish()
	})
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

	selfStateWorker, _ := NewSelfCheckWorker(database, logger, conf, notif)

	Convey("Go routine run before first send, should send after 10 seconds next time", t, func() {
		err := fmt.Errorf("DataBase doesn't work")
		shutdown := make(chan bool)
		var runWG sync.WaitGroup

		database.EXPECT().GetMetricsCount().Return(int64(1), nil).Times(11)
		database.EXPECT().GetChecksCount().Return(int64(1), err).Times(11)
		notif.EXPECT().Send(gomock.Any(), gomock.Any())
		runWG.Add(1)
		go selfStateWorker.Run(shutdown, &runWG)
		time.Sleep(time.Second*11 + time.Millisecond*500)
		close(shutdown)
		runWG.Wait()
	})
}

func configureWorker(t *testing.T) (needRun bool, selfStateWorker SelfCheckWorker, database *mock_moira_alert.MockDatabase, notif *mock_notifier.MockNotifier, conf Config, mockCtrl *gomock.Controller) {
	adminContact := map[string]string{
		"type":  "admin-mail",
		"value": "admin@company.com",
	}
	defaultCheckInterval = time.Second * 1
	conf = Config{
		Enabled: true,
		Contacts: []map[string]string{
			adminContact,
		},
		RedisDisconnectDelay:    10,
		LastMetricReceivedDelay: 60,
		LastCheckDelay:          120,
		NoticeInterval:          60,
	}

	mockCtrl = gomock.NewController(t)
	database = mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("SelfState")
	notif = mock_notifier.NewMockNotifier(mockCtrl)
	senders := map[string]bool{
		"admin-mail": true,
	}
	notif.EXPECT().GetSenders().Return(senders)

	selfState1, needRun := NewSelfCheckWorker(database, logger, conf, notif)
	selfStateWorker = *selfState1
	return
}

func configureNotificationPackage(adminContact map[string]string, errorValue int64, currentValue int64, message string) notifier.NotificationPackage {
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
				Value:     float64(currentValue),
			},
		},
		DontResend: true,
	}
}
