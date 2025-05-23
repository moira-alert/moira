package notifications

import (
	"testing"
	"time"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/moira-alert/moira/metrics"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_notifier "github.com/moira-alert/moira/mock/notifier"
	notifier2 "github.com/moira-alert/moira/notifier"
)

var notifierMetrics = metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")

func TestProcessScheduledEvent(t *testing.T) {
	subID2 := "subscriptionID-00000000000002"
	subID5 := "subscriptionID-00000000000005"
	subID7 := "subscriptionID-00000000000007"

	notification1 := moira.ScheduledNotification{
		Event: moira.NotificationEvent{
			SubscriptionID: &subID5,
			State:          moira.StateTEST,
		},
		Contact:   contact1,
		Throttled: false,
		Timestamp: 1441188915,
	}
	notification2 := moira.ScheduledNotification{
		Event: moira.NotificationEvent{
			SubscriptionID: &subID7,
			State:          moira.StateTEST,
			TriggerID:      "triggerID-00000000000001",
		},
		Contact:   contact2,
		Throttled: false,
		SendFail:  0,
		Timestamp: 1441188915,
	}
	notification3 := moira.ScheduledNotification{
		Event: moira.NotificationEvent{
			SubscriptionID: &subID2,
			State:          moira.StateTEST,
			TriggerID:      "triggerID-00000000000001",
		},
		Contact:   contact2,
		Throttled: false,
		SendFail:  0,
		Timestamp: 1441188915,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	notifier := mock_notifier.NewMockNotifier(mockCtrl)
	logger, _ := logging.GetLogger("Notification")
	worker := &FetchNotificationsWorker{
		Database: dataBase,
		Logger:   logger,
		Notifier: notifier,
		Metrics:  notifierMetrics,
	}

	Convey("Two different notifications, should send two packages", t, func() {
		dataBase.EXPECT().FetchNotifications(gomock.Any(), notifier2.NotificationsLimitUnlimited).Return([]*moira.ScheduledNotification{
			&notification1,
			&notification2,
		}, nil)

		pkg1 := notifier2.NotificationPackage{
			Trigger:    notification1.Trigger,
			Throttled:  notification1.Throttled,
			Contact:    notification1.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira.NotificationEvent{
				notification1.Event,
			},
		}
		pkg2 := notifier2.NotificationPackage{
			Trigger:    notification2.Trigger,
			Throttled:  notification2.Throttled,
			Contact:    notification2.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira.NotificationEvent{
				notification2.Event,
			},
		}

		dataBase.EXPECT().PushContactNotificationToHistory(&notification1).Return(nil).AnyTimes()
		dataBase.EXPECT().PushContactNotificationToHistory(&notification2).Return(nil).AnyTimes()
		notifier.EXPECT().Send(&pkg1, gomock.Any())
		notifier.EXPECT().Send(&pkg2, gomock.Any())
		notifier.EXPECT().GetReadBatchSize().Return(notifier2.NotificationsLimitUnlimited)
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}, nil)

		err := worker.processScheduledNotifications()
		So(err, ShouldBeEmpty)
	})

	Convey("Two same notifications, should send one package", t, func() {
		dataBase.EXPECT().FetchNotifications(gomock.Any(), notifier2.NotificationsLimitUnlimited).Return([]*moira.ScheduledNotification{ //nolint
			&notification2,
			&notification3,
		}, nil)

		pkg := notifier2.NotificationPackage{
			Trigger:    notification2.Trigger,
			Throttled:  notification2.Throttled,
			Contact:    notification2.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira.NotificationEvent{
				notification2.Event,
				notification3.Event,
			},
		}

		dataBase.EXPECT().PushContactNotificationToHistory(&notification2).Return(nil).AnyTimes()
		dataBase.EXPECT().PushContactNotificationToHistory(&notification3).Return(nil).AnyTimes()
		notifier.EXPECT().Send(&pkg, gomock.Any())
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{
			State: moira.SelfStateOK,
			Actor: moira.SelfStateActorManual,
		}, nil)
		notifier.EXPECT().GetReadBatchSize().Return(notifier2.NotificationsLimitUnlimited)

		err := worker.processScheduledNotifications()
		So(err, ShouldBeEmpty)
	})
}

func TestGoRoutine(t *testing.T) {
	subID5 := "subscriptionID-00000000000005"

	notification1 := moira.ScheduledNotification{
		Event: moira.NotificationEvent{
			SubscriptionID: &subID5,
			State:          moira.StateTEST,
		},
		Contact:   contact1,
		Throttled: false,
		Timestamp: 1441188915,
	}

	pkg := notifier2.NotificationPackage{
		Trigger:    notification1.Trigger,
		Throttled:  notification1.Throttled,
		Contact:    notification1.Contact,
		DontResend: false,
		FailCount:  0,
		Events: []moira.NotificationEvent{
			notification1.Event,
		},
	}

	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	notifier := mock_notifier.NewMockNotifier(mockCtrl)
	logger, _ := logging.GetLogger("Notification")

	worker := &FetchNotificationsWorker{
		Database: dataBase,
		Logger:   logger,
		Notifier: notifier,
		Metrics:  notifierMetrics,
	}

	shutdown := make(chan struct{})

	dataBase.EXPECT().FetchNotifications(gomock.Any(), notifier2.NotificationsLimitUnlimited).Return([]*moira.ScheduledNotification{&notification1}, nil)
	dataBase.EXPECT().PushContactNotificationToHistory(&notification1).Return(nil).AnyTimes()
	notifier.EXPECT().Send(&pkg, gomock.Any()).Do(func(arg0, arg1 interface{}) { close(shutdown) })
	notifier.EXPECT().StopSenders()
	notifier.EXPECT().GetReadBatchSize().Return(notifier2.NotificationsLimitUnlimited)
	dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{
		State: moira.SelfStateOK,
		Actor: moira.SelfStateActorManual,
	}, nil)

	worker.Start()
	waitTestEnd(shutdown, worker)
	mockCtrl.Finish()
}

func waitTestEnd(shutdown chan struct{}, worker *FetchNotificationsWorker) {
	select {
	case <-shutdown:
		worker.Stop() //nolint
		break
	case <-time.After(time.Second * 10):
		close(shutdown)
		break
	}
}

var contact1 = moira.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "email",
	Value: "mail1@example.com",
}

var contact2 = moira.ContactData{
	ID:    "ContactID-000000000000006",
	Type:  "unknown",
	Value: "no matter",
}
