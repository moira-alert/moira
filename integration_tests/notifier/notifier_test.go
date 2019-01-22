package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/op/go-logging"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/events"
	"github.com/moira-alert/moira/notifier/notifications"
)

var senderSettings = map[string]string{
	"type": "mega-sender",
}

var location, _ = time.LoadLocation("UTC")
var dateTimeFormat = "15:04 02.01.2006"

var notifierConfig = notifier.Config{
	SendingTimeout:   time.Millisecond * 10,
	ResendingTimeout: time.Hour * 24,
	Location:         location,
	DateTimeFormat:   dateTimeFormat,
}

var shutdown = make(chan struct{})

var notifierMetrics = metrics.ConfigureNotifierMetrics("notifier")
var logger, _ = logging.GetLogger("Notifier_Test")
var mockCtrl *gomock.Controller

var contact = moira.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "mega-sender",
	Value: "mail1@example.com",
}

var subscription = moira.SubscriptionData{
	ID:                "subscriptionID-00000000000001",
	Enabled:           true,
	Tags:              []string{"test-tag-1"},
	Contacts:          []string{contact.ID},
	ThrottlingEnabled: true,
}

var trigger = moira.Trigger{
	ID:      "triggerID-0000000000001",
	Name:    "test trigger 1",
	Targets: []string{"test.target.1"},
	Tags:    []string{"test-tag-1"},
}

var triggerData = moira.TriggerData{
	ID:      "triggerID-0000000000001",
	Name:    "test trigger 1",
	Targets: []string{"test.target.1"},
	Tags:    []string{"test-tag-1"},
}

var event = moira.NotificationEvent{
	Metric:    "generate.event.1",
	State:     "OK",
	OldState:  "WARN",
	TriggerID: "triggerID-0000000000001",
}

func TestNotifier(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()
	database := redis.NewDatabase(logger, redis.Config{Port: "6379", Host: "localhost"}, redis.Notifier)
	metricsSourceProvider := metricSource.CreateMetricSourceProvider(local.Create(database), nil)
	database.SaveContact(&contact)
	database.SaveSubscription(&subscription)
	database.SaveTrigger(trigger.ID, &trigger)
	database.PushNotificationEvent(&event, true)
	notifier2 := notifier.NewNotifier(database, logger, notifierConfig, notifierMetrics, metricsSourceProvider)
	sender := mock_moira_alert.NewMockSender(mockCtrl)
	sender.EXPECT().Init(senderSettings, logger, location, dateTimeFormat).Return(nil)
	notifier2.RegisterSender(senderSettings, sender)
	sender.EXPECT().SendEvents(gomock.Any(), contact, triggerData, gomock.Any(), false).Return(nil).Do(func(f ...interface{}) {
		fmt.Print("SendEvents called. End test")
		close(shutdown)
	})

	fetchEventsWorker := events.FetchEventsWorker{
		Database:  database,
		Logger:    logger,
		Metrics:   notifierMetrics,
		Scheduler: notifier.NewScheduler(database, logger, notifierMetrics),
	}

	fetchNotificationsWorker := notifications.FetchNotificationsWorker{
		Database: database,
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
		fmt.Print("Test timeout")
		close(shutdown)
		break
	}
}
