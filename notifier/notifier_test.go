package notifier

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metric_source/local"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

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
	t.Run("Test package with trigger events", func(t *testing.T) {
		expected := []string{"metricName1", "metricName2", "metricName3", "metricName5"}
		actual := notificationsPackage.GetMetricNames()
		require.Equal(t, expected, actual)
	})

	t.Run("Test package with no trigger events", func(t *testing.T) {
		pkg := NotificationPackage{}

		for _, event := range notificationsPackage.Events {
			if event.IsTriggerEvent {
				event.IsTriggerEvent = false
			}

			pkg.Events = append(pkg.Events, event)
		}

		expected := []string{"metricName1", "metricName2", "metricName3", "metricName4", "metricName5"}
		actual := pkg.GetMetricNames()
		require.Equal(t, expected, actual)
	})

	t.Run("Test empty notification package", func(t *testing.T) {
		emptyNotificationPackage := NotificationPackage{}
		actual := emptyNotificationPackage.GetMetricNames()
		require.Empty(t, actual)
	})
}

func TestGetWindow(t *testing.T) {
	t.Run("Test non-empty notification package", func(t *testing.T) {
		from, to, err := notificationsPackage.GetWindow()
		require.NoError(t, err)
		require.Equal(t, int64(11), from)
		require.Equal(t, int64(179), to)
	})

	t.Run("Test empty notification package", func(t *testing.T) {
		emptyNotificationPackage := NotificationPackage{}
		_, _, err := emptyNotificationPackage.GetWindow()
		require.EqualError(t, err, "not enough data to resolve package window")
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

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(fmt.Errorf("Can't send"))
	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)
	dataBase.EXPECT().UpdateContactScores([]string{pkg.Contact.ID}, gomock.Any()).DoAndReturn(func(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
		expected := moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     1,
			SuccessTXCount: 0,
			LastErrorMsg:   "Can't send",
			Status:         moira.ContactStatusFailed,
		}
		actual := updater(moira.ContactScore{})

		if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(moira.ContactScore{}, "LastErrorTimestamp")); diff != "" {
			t.Errorf("Not equal: %s", diff)
		}

		return nil
	})

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
	dataBase.EXPECT().UpdateContactScores([]string{pkg.Contact.ID}, gomock.Any()).DoAndReturn(func(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
		expected := moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     1,
			SuccessTXCount: 0,
			LastErrorMsg:   "some sender reason",
			Status:         moira.ContactStatusFailed,
		}
		actual := updater(moira.ContactScore{})

		if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(moira.ContactScore{}, "LastErrorTimestamp")); diff != "" {
			t.Errorf("Not equal: %s", diff)
		}

		return nil
	})

	var wg sync.WaitGroup

	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestSetContactScoreIfSuccessSending(t *testing.T) {
	configureNotifier(t, defaultConfig)

	defer afterTest()

	eventsData := []moira.NotificationEvent{event}
	pkg := NotificationPackage{
		Events: eventsData,
		Contact: moira.ContactData{
			Type: "test_contact_type",
		},
	}

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(nil)
	dataBase.EXPECT().UpdateContactScores([]string{pkg.Contact.ID}, gomock.Any()).DoAndReturn(func(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
		expected := moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     1,
			SuccessTXCount: 1,
			Status:         moira.ContactStatusOK,
		}
		actual := updater(moira.ContactScore{})

		if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(moira.ContactScore{}, "LastErrorTimestamp")); diff != "" {
			t.Errorf("Not equal: %s", diff)
		}

		return nil
	})

	var wg sync.WaitGroup

	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestSetContactScoreIfFailedSenging(t *testing.T) {
	configureNotifier(t, defaultConfig)

	defer afterTest()

	eventsData := []moira.NotificationEvent{event}
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

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(fmt.Errorf("some sender reason"))
	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)
	dataBase.EXPECT().UpdateContactScores([]string{pkg.Contact.ID}, gomock.Any()).DoAndReturn(func(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
		expected := moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     21,
			SuccessTXCount: 20,
			LastErrorMsg:   "some sender reason",
			Status:         moira.ContactStatusFailed,
		}
		actual := updater(moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     20,
			SuccessTXCount: 20,
		})

		if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(moira.ContactScore{}, "LastErrorTimestamp")); diff != "" {
			t.Errorf("Not equal: %s", diff)
		}

		return nil
	})

	var wg sync.WaitGroup

	standardNotifier.Send(&pkg, &wg)
	wg.Wait()
	time.Sleep(time.Second * 2)
}

func TestDropContactStatisticsOnOverflow(t *testing.T) {
	configureNotifier(t, defaultConfig)

	defer afterTest()

	eventsData := []moira.NotificationEvent{event}
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

	sender.EXPECT().SendEvents(eventsData, pkg.Contact, pkg.Trigger, plots, pkg.Throttled).Return(fmt.Errorf("some sender reason"))
	scheduler.EXPECT().ScheduleNotification(params, gomock.Any()).Return(&notification)
	dataBase.EXPECT().AddNotification(&notification).Return(nil)
	dataBase.EXPECT().UpdateContactScores([]string{pkg.Contact.ID}, gomock.Any()).DoAndReturn(func(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
		expected := moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     1,
			SuccessTXCount: 0,
			LastErrorMsg:   "some sender reason",
			Status:         moira.ContactStatusFailed,
		}
		actual := updater(moira.ContactScore{
			ContactID:      pkg.Contact.ID,
			AllTXCount:     math.MaxUint64,
			SuccessTXCount: 20,
		})

		if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(moira.ContactScore{}, "LastErrorTimestamp")); diff != "" {
			t.Errorf("Not equal: %s", diff)
		}

		return nil
	})

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
		fmt.Println("Trying to send for 10 second")
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
	dataBase.EXPECT().UpdateContactScores(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

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
	t.Helper()

	metricsRegistry, err := metrics.NewMetricContext(context.Background()).CreateRegistry()
	require.NoError(t, err)
	notifierMetrics, err := metrics.ConfigureNotifierMetrics(metrics.NewDummyRegistry(), metricsRegistry, "notifier")
	require.NoError(t, err)

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

	err = standardNotifier.RegisterSender(senderSettings, sender)

	require.NoError(t, err)
	require.Equal(t, map[string]bool{"test_contact_type": true}, standardNotifier.GetSenders())
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
