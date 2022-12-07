package notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/events"
	"github.com/moira-alert/moira/notifier/notifications"
	. "github.com/smartystreets/goconvey/convey"
)

var senderSettings = map[string]string{
	"type": "mega-sender",
}

var location, _ = time.LoadLocation("UTC")
var dateTimeFormat = "15:04 02.01.2006"

var notifierConfig = Config{
	SendingTimeout:   time.Millisecond * 10,
	ResendingTimeout: time.Hour * 24,
	Location:         location,
	DateTimeFormat:   dateTimeFormat,
	ReadBatchSize:    NotificationsLimitUnlimited,
}

var shutdown = make(chan struct{})

var notifierMetrics = metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")
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
	State:     moira.StateOK,
	OldState:  moira.StateWARN,
	TriggerID: "triggerID-0000000000001",
}

func TestNotifier(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("TestNotifier", t, func() {
		database := redis.NewTestDatabase(logger)
		metricsSourceProvider := metricSource.CreateMetricSourceProvider(local.Create(database), nil)
		err := database.SaveContact(&contact)
		So(err, ShouldBeNil)
		err = database.SaveSubscription(&subscription)
		So(err, ShouldBeNil)
		err = database.SaveTrigger(trigger.ID, &trigger)
		So(err, ShouldBeNil)
		err = database.PushNotificationEvent(&event, true)
		So(err, ShouldBeNil)

		notifier := NewNotifier(database, logger, notifierConfig, notifierMetrics, metricsSourceProvider, map[string]moira.ImageStore{})
		sender := mock_moira_alert.NewMockSender(mockCtrl)
		err = notifier.RegisterSender(senderSettings, sender)
		So(err, ShouldBeNil)
		sender.EXPECT().SendEvents(gomock.Any(), contact, triggerData, gomock.Any(), false).Return(nil).Do(func(arg0, arg1, arg2, arg3, arg4 interface{}) {
			fmt.Print("SendEvents called. End test")
			close(shutdown)
		})

		fetchEventsWorker := events.FetchEventsWorker{
			Database:  database,
			Logger:    logger,
			Metrics:   notifierMetrics,
			Scheduler: NewScheduler(database, logger, notifierMetrics),
		}

		fetchNotificationsWorker := notifications.FetchNotificationsWorker{
			Database: database,
			Logger:   logger,
			Notifier: notifier,
		}

		fetchEventsWorker.Start()
		fetchNotificationsWorker.Start()

		waitTestEnd()

		err = fetchEventsWorker.Stop()
		So(err, ShouldBeNil)
		err = fetchNotificationsWorker.Stop()
		So(err, ShouldBeNil)
	})
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
