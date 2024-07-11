package notifier

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	mock_scheduler "github.com/moira-alert/moira/mock/scheduler"
)

const (
	dateTimeFormat = "15:04 02.01.2006"
)

var (
	mockCtrl         *gomock.Controller
	sender           *mock_moira_alert.MockSender
	standardNotifier *StandardNotifier
	scheduler        *mock_scheduler.MockScheduler
	dataBase         *mock_moira_alert.MockDatabase
	logger           moira.Logger

	plots         [][]byte
	shutdown      = make(chan struct{})
	location, _   = time.LoadLocation("UTC")
	defaultConfig = Config{
		SendingTimeout:    10 * time.Millisecond,
		ResendingTimeout:  time.Hour * 24,
		ReschedulingDelay: time.Minute,
		Location:          location,
		DateTimeFormat:    dateTimeFormat,
		Senders: []map[string]interface{}{
			{
				"sender_type":  "test_sender_type",
				"contact_type": "test_contact_type",
			},
		},
	}
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
	configureNotifier(t, defaultConfig)
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "unknown contact",
		},
	}
	params := moira.SchedulerParams{
		Event:        event,
		Trigger:      pkg.Trigger,
		Contact:      pkg.Contact,
		Plotting:     pkg.Plotting,
		ThrottledOld: pkg.Throttled,
		SendFail:     pkg.FailCount + 1,
	}
	notification := moira.ScheduledNotification{}

	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
}

func TestFailSendEvent(t *testing.T) {
	configureNotifier(t, defaultConfig)
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test_contact_type",
		},
	}
	params := moira.SchedulerParams{
		Event:        event,
		Trigger:      pkg.Trigger,
		Contact:      pkg.Contact,
		Plotting:     pkg.Plotting,
		ThrottledOld: pkg.Throttled,
		SendFail:     pkg.FailCount + 1,
	}
	notification := moira.ScheduledNotification{}

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(fmt.Errorf("Cant't send"))
	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)

	var wg sync.WaitGroup
	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestNoResendForSendToBrokenContact(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	configureNotifier(t, defaultConfig)
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test_contact_type",
		},
	}
	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).
		Return(moira.NewSenderBrokenContactError(fmt.Errorf("some sender reason")))

	var wg sync.WaitGroup
	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestTimeout(t *testing.T) {
	configureNotifier(t, defaultConfig)
	var wg sync.WaitGroup
	defer afterTest()

	var eventsData moira.NotificationEvents = []moira.NotificationEvent{event}

	// Configure events with long sending time
	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test_contact_type",
		},
	}

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(nil).Do(func(arg0, arg1, arg2, arg3, arg4 interface{}) {
		fmt.Print("Trying to send for 10 second")
		time.Sleep(time.Second * 10)
	}).Times(maxParallelSendsPerSender)

	for i := 0; i < maxParallelSendsPerSender; i++ {
		standardNotifier.Send(&pkg, &wg)
		wg.Wait()
	}

	// Configure timeouted event
	notification := moira.ScheduledNotification{}
	pkg2 := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type:  "test_contact_type",
			Value: "fail contact",
		},
	}
	params := moira.SchedulerParams{
		Event:        event,
		Trigger:      pkg2.Trigger,
		Contact:      pkg2.Contact,
		Plotting:     pkg2.Plotting,
		ThrottledOld: pkg2.Throttled,
		SendFail:     pkg2.FailCount + 1,
	}

	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil).Do(func(f ...interface{}) { close(shutdown) })

	standardNotifier.Send(&pkg2, &wg)
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

func configureNotifier(t *testing.T, config Config) {
	notifierMetrics := metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), "notifier")

	mockCtrl = gomock.NewController(t)
	dataBase = mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ = logging.GetLogger("Scheduler")
	scheduler = mock_scheduler.NewMockScheduler(mockCtrl)
	sender = mock_moira_alert.NewMockSender(mockCtrl)
	metricsSourceProvider := metricSource.CreateTestMetricSourceProvider(local.Create(dataBase), nil, nil)
	systemClock := mock_clock.NewMockClock(mockCtrl)

	schedulerConfig := SchedulerConfig{ReschedulingDelay: config.ReschedulingDelay}

	standardNotifier = NewNotifier(
		dataBase,
		logger,
		config,
		notifierMetrics,
		metricsSourceProvider,
		map[string]moira.ImageStore{},
		systemClock,
		schedulerConfig,
	)
	standardNotifier.scheduler = scheduler
	senderSettings := map[string]interface{}{
		"sender_type":  "test_type",
		"contact_type": "test_contact_type",
	}

	sender.EXPECT().Init(senderSettings, logger, location, dateTimeFormat).Return(nil)

	err := standardNotifier.RegisterSender(senderSettings, sender)

	Convey("Should return one sender", t, func() {
		So(err, ShouldBeNil)
		So(standardNotifier.GetSenders(), ShouldResemble, map[string]bool{"test_contact_type": true})
	})
}

func afterTest() {
	mockCtrl.Finish()
	standardNotifier.StopSenders()
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
