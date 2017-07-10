package notifications

import (
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/mock/notifier"
	notifier2 "github.com/moira-alert/moira-alert/notifier"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestProcessScheduledEvent(t *testing.T) {
	notification1 := moira_alert.ScheduledNotification{
		Event: moira_alert.EventData{
			SubscriptionID: "subscriptionID-00000000000005",
			State:          "TEST",
		},
		Contact:   contact1,
		Throttled: false,
		Timestamp: 1441188915,
	}
	notification2 := moira_alert.ScheduledNotification{
		Event: moira_alert.EventData{
			SubscriptionID: "subscriptionID-00000000000007",
			State:          "TEST",
			TriggerID:      "triggerID-00000000000001",
		},
		Contact:   contact2,
		Throttled: false,
		SendFail:  0,
		Timestamp: 1441188915,
	}
	notification3 := moira_alert.ScheduledNotification{
		Event: moira_alert.EventData{
			SubscriptionID: "subscriptionID-00000000000002",
			State:          "TEST",
			TriggerID:      "triggerID-00000000000001",
		},
		Contact:   contact2,
		Throttled: false,
		SendFail:  0,
		Timestamp: 1441188915,
	}

	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	notifier := mock_notifier.NewMockNotifier(mockCtrl)
	logger, _ := logging.GetLogger("Notification")

	worker := Init(dataBase, logger, notifier)
	Convey("Two different notifications, should send two packages", t, func() {
		dataBase.EXPECT().GetNotifications(gomock.Any()).Return([]*moira_alert.ScheduledNotification{
			&notification1,
			&notification2,
		}, nil)

		pkg1 := notifier2.NotificationPackage{
			Trigger:    notification1.Trigger,
			Throttled:  notification1.Throttled,
			Contact:    notification1.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira_alert.EventData{
				notification1.Event,
			},
		}
		pkg2 := notifier2.NotificationPackage{
			Trigger:    notification2.Trigger,
			Throttled:  notification2.Throttled,
			Contact:    notification2.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira_alert.EventData{
				notification2.Event,
			},
		}
		notifier.EXPECT().Send(&pkg1, gomock.Any())
		notifier.EXPECT().Send(&pkg2, gomock.Any())
		err := worker.processScheduledNotifications()
		So(err, ShouldBeEmpty)
		mockCtrl.Finish()
	})

	Convey("Two same notifications, should send one package", t, func() {
		dataBase.EXPECT().GetNotifications(gomock.Any()).Return([]*moira_alert.ScheduledNotification{
			&notification2,
			&notification3,
		}, nil)

		pkg := notifier2.NotificationPackage{
			Trigger:    notification2.Trigger,
			Throttled:  notification2.Throttled,
			Contact:    notification2.Contact,
			DontResend: false,
			FailCount:  0,
			Events: []moira_alert.EventData{
				notification2.Event,
				notification3.Event,
			},
		}

		notifier.EXPECT().Send(&pkg, gomock.Any())
		err := worker.processScheduledNotifications()
		So(err, ShouldBeEmpty)
		mockCtrl.Finish()
	})
}

var contact1 = moira_alert.ContactData{
	ID:    "ContactID-000000000000001",
	Type:  "email",
	Value: "mail1@example.com",
}

var contact2 = moira_alert.ContactData{
	ID:    "ContactID-000000000000006",
	Type:  "unknown",
	Value: "no matter",
}
