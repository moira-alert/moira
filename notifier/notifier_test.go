package notifier

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_scheduler "github.com/moira-alert/moira/mock/scheduler"
)

var (
	plots    [][]byte
	shutdown = make(chan struct{})
)

var (
	mockCtrl  *gomock.Controller
	sender    *mock_moira_alert.MockSender
	notif     *StandardNotifier
	scheduler *mock_scheduler.MockScheduler
	dataBase  *mock_moira_alert.MockDatabase
	logger    moira.Logger
)

func TestGetMetricNames(t *testing.T) {
	Convey("Test non-empty notification package", t, func() {
		Convey("Test package with trigger events", func() {
			expected := []string{"metricName1", "metricName2", "metricName3", "metricName5"}
			actual := notificationsPackage.GetMetricNames()
			So(actual, ShouldResemble, expected)
		})
		Convey("Test package with no trigger events", func() {
			pkg := NotificationPackage{}
			for _, event := range notificationsPackage.Events {
				if event.IsTriggerEvent {
					event.IsTriggerEvent = false
				}
				pkg.Events = append(pkg.Events, event)
			}
			expected := []string{"metricName1", "metricName2", "metricName3", "metricName4", "metricName5"}
			actual := pkg.GetMetricNames()
			So(actual, ShouldResemble, expected)
		})
	})
	Convey("Test empty notification package", t, func() {
		emptyNotificationPackage := NotificationPackage{}
		actual := emptyNotificationPackage.GetMetricNames()
		So(actual, ShouldHaveLength, 0)
	})
}

func TestGetWindow(t *testing.T) {
	Convey("Test non-empty notification package", t, func() {
		from, to, err := notificationsPackage.GetWindow()
		So(err, ShouldBeNil)
		So(from, ShouldEqual, 11)
		So(to, ShouldEqual, 179)
	})
	Convey("Test empty notification package", t, func() {
		emptyNotificationPackage := NotificationPackage{}
		_, _, err := emptyNotificationPackage.GetWindow()
		So(err, ShouldResemble, fmt.Errorf("not enough data to resolve package window"))
	})
}

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
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg.Trigger, pkg.Contact, pkg.Plotting, pkg.Throttled, pkg.FailCount+1).Return(&notification)
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
	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(fmt.Errorf("Cant't send"))
	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg.Trigger, pkg.Contact, pkg.Plotting, pkg.Throttled, pkg.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	notif.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestTimeout(t *testing.T) {
	configureNotifier(t)
	var wg sync.WaitGroup
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	// Configure events with long sending time
	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test",
		},
	}

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(nil).Do(func(f ...interface{}) {
		fmt.Print("Trying to send for 10 second")
		time.Sleep(time.Second * 10)
	}).Times(maxParallelSendsPerSender)

	for i := 0; i < maxParallelSendsPerSender; i++ {
		notif.Send(&pkg, &wg)
		wg.Wait()
	}

	// Configure timeouted event
	notification := moira.ScheduledNotification{}
	pkg2 := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type:  "test",
			Value: "fail contact",
		},
	}

	scheduler.EXPECT().ScheduleNotification(gomock.Any(), event, pkg2.Trigger, pkg2.Contact, pkg.Plotting, pkg2.Throttled, pkg2.FailCount+1).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil).Do(func(f ...interface{}) { close(shutdown) })

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
	notifierMetrics := metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")
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
	metricsSourceProvider := metricSource.CreateMetricSourceProvider(local.Create(dataBase), nil)

	notif = NewNotifier(dataBase, logger, config, notifierMetrics, metricsSourceProvider, map[string]moira.ImageStore{})
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
	State:          moira.StateOK,
	OldState:       moira.StateWARN,
	TriggerID:      "triggerID-0000000000001",
	SubscriptionID: &subID,
}

var notificationsPackage = NotificationPackage{
	Events: []moira.NotificationEvent{
		{Metric: "metricName1", Timestamp: 15, IsTriggerEvent: false},
		{Metric: "metricName2", Timestamp: 11, IsTriggerEvent: false},
		{Metric: "metricName3", Timestamp: 31, IsTriggerEvent: false},
		{Metric: "metricName4", Timestamp: 179, IsTriggerEvent: true},
		{Metric: "metricName5", Timestamp: 12, IsTriggerEvent: false},
	},
}
