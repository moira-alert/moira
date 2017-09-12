package events

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira/mock/moira-alert"
	mock_scheduler "github.com/moira-alert/moira/mock/scheduler"
	"github.com/moira-alert/moira/notifier"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

var metrics2 = metrics.ConfigureNotifierMetrics("notifier")

func TestEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Events")

	worker := FetchEventsWorker{
		Database:  dataBase,
		Logger:    logger,
		Metrics:   metrics2,
		Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
	}

	Convey("When event is TEST and subscription is disabled, should add new notification", t, func() {
		event := moira.NotificationEvent{
			State:          "TEST",
			SubscriptionID: &subscription.ID,
		}
		dataBase.EXPECT().GetSubscription(*event.SubscriptionID).Times(1).Return(subscription, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		notification := moira.ScheduledNotification{
			Event: moira.NotificationEvent{
				TriggerID:      event.TriggerID,
				State:          event.State,
				OldState:       event.OldState,
				Metric:         event.Metric,
				SubscriptionID: event.SubscriptionID,
			},
			SendFail:  0,
			Timestamp: time.Now().Unix(),
			Throttled: false,
			Contact:   contact,
		}
		dataBase.EXPECT().AddNotification(&notification).Times(1)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestNoSubscription(t *testing.T) {
	Convey("When no subscription by event tags, should not call AddNotification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Events")

		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		dataBase.EXPECT().GetTagsSubscriptions(append(triggerData.Tags, event.GetEventTags()...)).Times(1).Return(make([]*moira.SubscriptionData, 0), nil)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestDisabledNotification(t *testing.T) {
	Convey("When subscription event tags is disabled, should not call AddNotification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)

		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&disabledSubscription}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Subscription %s is disabled", disabledSubscription.ID)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestExtraTags(t *testing.T) {
	Convey("When trigger has not all subscription tags, should not call AddNotification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)

		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&multipleTagsSubscription}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Subscription %s has extra tags", multipleTagsSubscription.ID)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestAddNotification(t *testing.T) {
	Convey("When good subscription, should add new notification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Events")
		scheduler := mock_scheduler.NewMockScheduler(mockCtrl)
		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: scheduler,
		}

		event := moira.NotificationEvent{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      triggerData.ID,
			SubscriptionID: &subscription.ID,
		}
		emptyNotification := moira.ScheduledNotification{}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&subscription}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, triggerData, contact, false, 0).Times(1).Return(&emptyNotification)
		dataBase.EXPECT().AddNotification(&emptyNotification).Times(1).Return(nil)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestAddOneNotificationByTwoSubscriptionsWithSame(t *testing.T) {
	Convey("When good subscription and create 2 same scheduled notifications, should add one new notification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Events")
		scheduler := mock_scheduler.NewMockScheduler(mockCtrl)
		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: scheduler,
		}

		event := moira.NotificationEvent{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      triggerData.ID,
			SubscriptionID: &subscription.ID,
		}
		event2 := event
		event2.SubscriptionID = &subscription4.ID

		notification2 := moira.ScheduledNotification{}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&subscription, &subscription4}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(2).Return(contact, nil)

		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, triggerData, contact, false, 0).Times(1).Return(&notification2)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event2, triggerData, contact, false, 0).Times(1).Return(&notification2)

		dataBase.EXPECT().AddNotification(&notification2).Times(1).Return(nil)

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestFailReadContact(t *testing.T) {
	Convey("When read contact returns error, should not call AddNotification and not crashed", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)
		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&subscription}, nil)
		getContactError := fmt.Errorf("Can not get contact")
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(moira.ContactData{}, getContactError)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Processing contact ids %v for subscription %s", subscription.Contacts, subscription.ID)
		logger.EXPECT().Warning(getContactError.Error())

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestEmptySubscriptions(t *testing.T) {
	Convey("When subscription is empty value object", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)
		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{{ThrottlingEnabled: true}}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Subscription %s is disabled", "")

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})

	Convey("When subscription is nil", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)
		worker := FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: notifier.NewScheduler(dataBase, logger, metrics2),
		}

		event := moira.NotificationEvent{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: triggerData.ID,
		}

		dataBase.EXPECT().GetTrigger(event.TriggerID).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{nil}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Subscription is nil")

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestGoRoutine(t *testing.T) {
	Convey("When good subscription, should add new notification", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Events")
		scheduler := mock_scheduler.NewMockScheduler(mockCtrl)

		worker := &FetchEventsWorker{
			Database:  dataBase,
			Logger:    logger,
			Metrics:   metrics2,
			Scheduler: scheduler,
		}

		event := moira.NotificationEvent{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      triggerData.ID,
			SubscriptionID: &subscription.ID,
		}
		emptyNotification := moira.ScheduledNotification{}
		shutdown := make(chan bool)

		dataBase.EXPECT().FetchNotificationEvent().Return(moira.NotificationEvent{}, fmt.Errorf("3433434")).Do(func(f ...interface{}) {
			dataBase.EXPECT().FetchNotificationEvent().Return(event, nil).Do(func(f ...interface{}) {
				dataBase.EXPECT().FetchNotificationEvent().AnyTimes().Return(moira.NotificationEvent{}, database.ErrNil)
			})
		})
		dataBase.EXPECT().GetTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		tags := append(triggerData.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]*moira.SubscriptionData{&subscription}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, triggerData, contact, false, 0).Times(1).Return(&emptyNotification)
		dataBase.EXPECT().AddNotification(&emptyNotification).Times(1).Return(nil).Do(func(f ...interface{}) { close(shutdown) })

		worker.Start()
		waitTestEnd(shutdown, worker)
	})
}

func waitTestEnd(shutdown chan bool, worker *FetchEventsWorker) {
	select {
	case <-shutdown:
		worker.Stop()
		break
	case <-time.After(time.Second * 10):
		close(shutdown)
		break
	}
}

var warnValue float64 = 10
var errorValue float64 = 20

var triggerData = moira.TriggerData{
	ID:         "triggerID-0000000000001",
	Name:       "test trigger",
	Targets:    []string{"test.target.5"},
	WarnValue:  warnValue,
	ErrorValue: errorValue,
	Tags:       []string{"test-tag"},
}

var trigger = moira.Trigger{
	ID:         "triggerID-0000000000001",
	Name:       "test trigger",
	Targets:    []string{"test.target.5"},
	WarnValue:  &warnValue,
	ErrorValue: &errorValue,
	Tags:       []string{"test-tag"},
}

var contact = moira.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "email",
	Value: "mail1@example.com",
}

var subscription = moira.SubscriptionData{
	ID:                "subscriptionID-00000000000001",
	Enabled:           true,
	Tags:              []string{"test-tag"},
	Contacts:          []string{contact.ID},
	ThrottlingEnabled: true,
}

var subscription4 = moira.SubscriptionData{
	ID:                "subscriptionID-00000000000004",
	Enabled:           true,
	Tags:              []string{"test-tag"},
	Contacts:          []string{contact.ID},
	ThrottlingEnabled: true,
}

var disabledSubscription = moira.SubscriptionData{
	ID:                "subscriptionID-00000000000002",
	Enabled:           false,
	Tags:              []string{"test-tag"},
	Contacts:          []string{contact.ID},
	ThrottlingEnabled: true,
}

var multipleTagsSubscription = moira.SubscriptionData{
	ID:                "subscriptionID-00000000000003",
	Enabled:           true,
	Tags:              []string{"test-tag", "one-more-tag"},
	Contacts:          []string{contact.ID},
	ThrottlingEnabled: true,
}
