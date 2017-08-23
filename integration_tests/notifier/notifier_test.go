package notifier

import (
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
	"github.com/moira-alert/moira-alert/notifier/events"
	"github.com/moira-alert/moira-alert/notifier/notifications"
	"github.com/op/go-logging"
	"sync"
	"testing"
	"time"
)

var senderSettings = map[string]string{
	"type": "mega-sender",
}

var notifierConfig = notifier.Config{
	SendingTimeout:   time.Millisecond * 10,
	ResendingTimeout: time.Hour * 24,
}

var shutdown = make(chan bool)
var waitGroup sync.WaitGroup

var metrics2 = metrics.ConfigureNotifierMetrics()
var logger, _ = logging.GetLogger("Notifier_Test")
var mockCtrl *gomock.Controller

func TestNotifier(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer afterTest()
	fakeDataBase := redis.NewFakeDatabase(logger)
	notifier2 := notifier.NewNotifier(fakeDataBase, logger, notifierConfig, metrics2)
	sender := mock_moira_alert.NewMockSender(mockCtrl)
	sender.EXPECT().Init(senderSettings, logger).Return(nil)
	notifier2.RegisterSender(senderSettings, sender)
	sender.EXPECT().SendEvents(gomock.Any(), contact, trigger, false).Return(nil).Do(func(f ...interface{}) {
		logger.Debugf("SendEvents called. End test")
		close(shutdown)
	})

	fetchEventsWorker := events.FetchEventsWorker{
		Database:  fakeDataBase,
		Logger:    logger,
		Metrics:   metrics2,
		Scheduler: notifier.NewScheduler(fakeDataBase, logger),
	}

	fetchNotificationsWorker := notifications.FetchNotificationsWorker{
		Database: fakeDataBase,
		Logger:   logger,
		Notifier: notifier2,
	}

	fetchEventsWorker.Start()
	fetchNotificationsWorker.Start()

	waitTestEnd()

	fetchEventsWorker.Stop()
	fetchNotificationsWorker.Stop()
}

func waitTestEnd() {
	select {
	case <-shutdown:
		break
	case <-time.After(time.Second * 30):
		logger.Debugf("Test timeout")
		close(shutdown)
		break
	}
}

func afterTest() {
	waitGroup.Wait()
	mockCtrl.Finish()
}

var trigger = moira.TriggerData{
	ID:         "triggerID-0000000000001",
	Name:       "test trigger 1",
	Targets:    []string{"test.target.1"},
	WarnValue:  10,
	ErrorValue: 20,
	Tags:       []string{"test-tag-1"},
}

var contact = moira.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "mega-sender",
	Value: "mail1@example.com",
}
