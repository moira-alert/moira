package notifier

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/moira-alert/moira-alert/mock/scheduler"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

var (
	mockCtrl  *gomock.Controller
	sender    *mock_moira_alert.MockSender
	notif     *StandardNotifier
	scheduler *mock_scheduler.MockScheduler
	dataBase  *mock_moira_alert.MockDatabase
)

func TestUnknownContactType(t *testing.T) {
	configureNotifier(t)
	defer afterTest()

	var eventsData moira_alert.EventsData = []moira_alert.EventData{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira_alert.ContactData{
			Type: "unknown contact",
		},
	}
	notification := moira_alert.ScheduledNotification{}
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg.Trigger, pkg.Contact, pkg.Throttled, pkg.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	wg.Wait()
}

func TestFailSendEvent(t *testing.T) {
	configureNotifier(t)
	defer afterTest()

	var eventsData moira_alert.EventsData = []moira_alert.EventData{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira_alert.ContactData{
			Type: "test",
		},
	}
	notification := moira_alert.ScheduledNotification{}
	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, pkg.Throttled).Return(fmt.Errorf("Cant't send"))
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg.Trigger, pkg.Contact, pkg.Throttled, pkg.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestTimeout(t *testing.T) {
	configureNotifier(t)
	defer afterTest()

	var eventsData moira_alert.EventsData = []moira_alert.EventData{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira_alert.ContactData{
			Type: "test",
		},
	}

	pkg2 := NotificationPackage{
		Events: eventsData,
		Contact: moira_alert.ContactData{
			Type:  "test",
			Value: "fail contact",
		},
	}
	notification := moira_alert.ScheduledNotification{}
	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, pkg.Throttled).Do(func(f ...interface{}) { time.Sleep(time.Second * 10) }).Return(nil)
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg2.Trigger, pkg2.Contact, pkg2.Throttled, pkg2.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	notif.Send(&pkg2, &wg)
	wg.Wait()
	time.Sleep(time.Second * 5)
}

func configureNotifier(t *testing.T) {
	go_metrics.ConfigureNotifierMetrics(graphite.Config{})
	config := Config{
		SendingTimeout:   time.Millisecond * 10,
		ResendingTimeout: time.Hour * 24,
	}

	mockCtrl = gomock.NewController(t)
	dataBase = mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Scheduler")
	scheduler = mock_scheduler.NewMockScheduler(mockCtrl)
	sender = mock_moira_alert.NewMockSender(mockCtrl)

	notif = Init(dataBase, logger, config)
	notif.scheduler = scheduler
	senderSettings := map[string]string{
		"type": "test",
	}

	sender.EXPECT().Init(senderSettings, logger).Return(nil)

	notif.RegisterSender(senderSettings, sender)

	Convey("Should return one sender", t, func() {
		So(notif.GetSenders(), ShouldResemble, map[string]bool{"test": true})
	})
}

func afterTest() {
	mockCtrl.Finish()
	notif.StopSenders()
}

var event = moira_alert.EventData{
	Metric:         "generate.event.1",
	State:          "OK",
	OldState:       "WARN",
	TriggerID:      "triggerID-0000000000001",
	SubscriptionID: "SubscriptionID-000000000000001",
}
