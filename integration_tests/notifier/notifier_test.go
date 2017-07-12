package test_notifier

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

var metrics = go_metrics.ConfigureNotifierMetrics()
var logger, _ = logging.GetLogger("Notifier_Test")
var mockCtrl gomock.Controller

func TestNotifier(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer afterTest()
	fakeDataBase := redis.InitFake(logger)
	notifier2 := notifier.Init(fakeDataBase, logger, notifierConfig, metrics)
	sender := mock_moira_alert.NewMockSender(mockCtrl)
	sender.EXPECT().Init(senderSettings, logger).Return(nil)
	notifier2.RegisterSender(senderSettings, sender)
	var eventsData moira_alert.EventsData = []moira_alert.EventData{event}
	sender.EXPECT().SendEvents(eventsData, contact, trigger, false).Return(nil).Do(func(f ...interface{}) {
		logger.Debugf("SendEvents called. End test")
		close(shutdown)
	})

	initWorkers(notifier2, logger, fakeDataBase)
	waitTestEnd()
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

func initWorkers(notifier2 notifier.Notifier, logger moira_alert.Logger, connector *redis.DbConnector) {
	fetchEventsWorker := events.Init(connector, logger, metrics)
	fetchNotificationsWorker := notifications.Init(connector, logger, notifier2)

	run(fetchEventsWorker, shutdown, &waitGroup)
	run(fetchNotificationsWorker, shutdown, &waitGroup)
}

func run(worker moira_alert.Worker, shutdown chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go worker.Run(shutdown, wg)
}

var event = moira_alert.EventData{
	Metric:         "generate.event.1",
	State:          "OK",
	OldState:       "WARN",
	TriggerID:      trigger.ID,
	SubscriptionID: "subscriptionID-00000000000001",
}

var trigger = moira_alert.TriggerData{
	ID:         "triggerID-0000000000001",
	Name:       "test trigger 1",
	Targets:    []string{"test.target.1"},
	WarnValue:  10,
	ErrorValue: 20,
	Tags:       []string{"test-tag-1"},
}

var contact = moira_alert.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "mega-sender",
	Value: "mail1@example.com",
}
