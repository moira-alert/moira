package events

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/mock/scheduler"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

var metrics2 = metrics.ConfigureNotifierMetrics()

func TestEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Events")

	worker := NewFetchEventWorker(dataBase, logger, metrics2)

	Convey("When event is TEST and subscription is disabled, should add new notification", t, func() {
		event := moira.EventData{
			State:          "TEST",
			SubscriptionID: subscription.ID,
		}
		dataBase.EXPECT().GetSubscription(event.SubscriptionID).Times(1).Return(subscription, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		notification := moira.ScheduledNotification{
			Event: moira.EventData{
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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)

		event := moira.EventData{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: trigger.ID,
		}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		dataBase.EXPECT().GetTagsSubscriptions(append(trigger.Tags, event.GetEventTags()...)).Times(1).Return(make([]moira.SubscriptionData, 0), nil)

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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)

		event := moira.EventData{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: trigger.ID,
		}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{disabledSubscription}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, event.Value, event.OldState, event.State)
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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)

		event := moira.EventData{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: trigger.ID,
		}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{multipleTagsSubscription}, nil)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, event.Value, event.OldState, event.State)
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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)
		worker.scheduler = scheduler

		event := moira.EventData{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      trigger.ID,
			SubscriptionID: subscription.ID,
		}
		emptyNotification := moira.ScheduledNotification{}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{subscription}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, trigger, contact, false, 0).Times(1).Return(&emptyNotification)
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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)
		worker.scheduler = scheduler

		event := moira.EventData{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      trigger.ID,
			SubscriptionID: subscription.ID,
		}
		event2 := event
		event2.SubscriptionID = subscription4.ID

		notification2 := moira.ScheduledNotification{}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{subscription, subscription4}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(2).Return(contact, nil)

		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, trigger, contact, false, 0).Times(1).Return(&notification2)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event2, trigger, contact, false, 0).Times(1).Return(&notification2)

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
		worker := NewFetchEventWorker(dataBase, logger, metrics2)

		event := moira.EventData{
			Metric:    "generate.event.1",
			State:     "OK",
			OldState:  "WARN",
			TriggerID: trigger.ID,
		}

		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{subscription}, nil)
		getContactError := fmt.Errorf("Can not get contact")
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(moira.ContactData{}, getContactError)

		logger.EXPECT().Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, event.Value, event.OldState, event.State)
		logger.EXPECT().Debugf("Getting subscriptions for tags %v", tags)
		logger.EXPECT().Debugf("Processing contact ids %v for subscription %s", subscription.Contacts, subscription.ID)
		logger.EXPECT().Warning(getContactError.Error())

		err := worker.processEvent(event)
		So(err, ShouldBeEmpty)
	})
}

func TestGoRoutine(t *testing.T) {
	Convey("When good subscription, should add new notification", t, func() {
		mockCtrl := gomock.NewController(t)
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		logger, _ := logging.GetLogger("Events")
		scheduler := mock_scheduler.NewMockScheduler(mockCtrl)
		worker := NewFetchEventWorker(dataBase, logger, metrics2)
		worker.scheduler = scheduler

		event := moira.EventData{
			Metric:         "generate.event.1",
			State:          "OK",
			OldState:       "WARN",
			TriggerID:      trigger.ID,
			SubscriptionID: subscription.ID,
		}
		emptyNotification := moira.ScheduledNotification{}

		dataBase.EXPECT().FetchEvent().Return(&event, nil)
		dataBase.EXPECT().FetchEvent().AnyTimes().Return(nil, nil)
		dataBase.EXPECT().GetNotificationTrigger(event.TriggerID).Times(1).Return(trigger, nil)
		dataBase.EXPECT().GetTriggerTags(event.TriggerID).Times(1).Return(trigger.Tags, nil)
		tags := append(trigger.Tags, event.GetEventTags()...)
		dataBase.EXPECT().GetTagsSubscriptions(tags).Times(1).Return([]moira.SubscriptionData{subscription}, nil)
		dataBase.EXPECT().GetContact(contact.ID).Times(1).Return(contact, nil)
		scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, trigger, contact, false, 0).Times(1).Return(&emptyNotification)
		dataBase.EXPECT().AddNotification(&emptyNotification).Times(1).Return(nil)

		shutdown := make(chan bool)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go worker.Run(shutdown, &wg)
		time.Sleep(time.Second * 5)
		close(shutdown)
		wg.Wait()
		mockCtrl.Finish()
	})
}

var trigger = moira.TriggerData{
	ID:         "triggerID-0000000000001",
	Name:       "test trigger",
	Targets:    []string{"test.target.5"},
	WarnValue:  10,
	ErrorValue: 20,
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
