package notifier

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite/go-metrics"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/moira-alert/moira/mock/scheduler"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

var shutdown = make(chan bool)

var (
	mockCtrl  *gomock.Controller
	sender    *mock_moira_alert.MockSender
	notif     *StandardNotifier
	scheduler *mock_scheduler.MockScheduler
	dataBase  *mock_moira_alert.MockDatabase
	logger    moira.Logger
)

func TestUnknownContactType(t *testing.T) {
	configureNotifier(t)
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "unknown contact",
		},
	}
	notification := moira.ScheduledNotification{}
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg.Trigger, pkg.Contact, pkg.Throttled, pkg.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	wg.Wait()
}

func TestFailSendEvent(t *testing.T) {
	configureNotifier(t)
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test",
		},
	}
	notification := moira.ScheduledNotification{}
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

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test",
		},
	}

	pkg2 := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type:  "test",
			Value: "fail contact",
		},
	}
	notification := moira.ScheduledNotification{}
	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, pkg.Throttled).Return(nil).Do(func(f ...interface{}) {
		fmt.Print("Trying to send for 10 second")
		time.Sleep(time.Second * 10)
	})
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg2.Trigger, pkg2.Contact, pkg2.Throttled, pkg2.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil).Do(func(f ...interface{}) { close(shutdown) })

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	wg.Wait()
	notif.Send(&pkg2, &wg)
	wg.Wait()
	waitTestEnd()
}

func waitTestEnd() {
	select {
	case <-shutdown:
		break
	case <-time.After(time.Second * 30):
		close(shutdown)
		break
	}
}

func configureNotifier(t *testing.T) {
	notifierMetrics := metrics.ConfigureNotifierMetrics("notifier")
	var location, _ = time.LoadLocation("UTC")
	dateTimeFormat := "15:04 02.01.2006"
	config := Config{
		SendingTimeout:   time.Millisecond * 10,
		ResendingTimeout: time.Hour * 24,
		Location:         location,
		DateTimeFormat:   dateTimeFormat,
	}

	mockCtrl = gomock.NewController(t)
	dataBase = mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ = logging.GetLogger("Scheduler")
	scheduler = mock_scheduler.NewMockScheduler(mockCtrl)
	sender = mock_moira_alert.NewMockSender(mockCtrl)

	notif = NewNotifier(dataBase, logger, config, notifierMetrics)
	notif.scheduler = scheduler
	senderSettings := map[string]string{
		"type": "test",
	}

	sender.EXPECT().Init(senderSettings, logger, location, "15:04 02.01.2006").Return(nil)

	notif.RegisterSender(senderSettings, sender)

	Convey("Should return one sender", t, func() {
		So(notif.GetSenders(), ShouldResemble, map[string]bool{"test": true})
	})
}

func afterTest() {
	mockCtrl.Finish()
	notif.StopSenders()
}

var subID = "SubscriptionID-000000000000001"

var event = moira.NotificationEvent{
	Metric:         "generate.event.1",
	State:          "OK",
	OldState:       "WARN",
	TriggerID:      "triggerID-0000000000001",
	SubscriptionID: &subID,
}
