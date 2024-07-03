package notifier

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/plotting"
)

// NotificationPackage represent sending data.
type NotificationPackage struct {
	Events     []moira.NotificationEvent
	Trigger    moira.TriggerData
	Contact    moira.ContactData
	Plotting   moira.PlottingData
	FailCount  int
	Throttled  bool
	DontResend bool
}

// String returns notification package summary.
func (pkg NotificationPackage) String() string {
	return fmt.Sprintf("package of %d notifications to %s", len(pkg.Events), pkg.Contact.Value)
}

// GetWindow returns the earliest and the latest notification package timestamps.
func (pkg NotificationPackage) GetWindow() (from, to int64, err error) {
	timeStamps := make([]int64, 0)
	for _, event := range pkg.Events {
		timeStamps = append(timeStamps, event.Timestamp)
	}
	if len(timeStamps) == 0 {
		return 0, 0, fmt.Errorf("not enough data to resolve package window")
	}
	from = timeStamps[0]
	to = timeStamps[len(timeStamps)-1]
	for _, timeStamp := range timeStamps {
		if timeStamp < from {
			from = timeStamp
		}
		if timeStamp > to {
			to = timeStamp
		}
	}
	return from, to, nil
}

// GetMetricNames returns all metric names found in package events.
func (pkg NotificationPackage) GetMetricNames() []string {
	metricNames := make([]string, 0)
	for _, event := range pkg.Events {
		if !event.IsTriggerEvent {
			metricNames = append(metricNames, event.Metric)
		}
	}
	return metricNames
}

// Notifier implements notification functionality.
type Notifier interface {
	Send(pkg *NotificationPackage, waitGroup *sync.WaitGroup)
	RegisterSender(senderSettings map[string]interface{}, sender moira.Sender) error
	StopSenders()
	GetSenders() map[string]bool
	GetReadBatchSize() int64
}

// StandardNotifier represent notification functionality.
type StandardNotifier struct {
	waitGroup            sync.WaitGroup
	senders              map[string]chan NotificationPackage
	logger               moira.Logger
	database             moira.Database
	scheduler            Scheduler
	config               Config
	metrics              *metrics.NotifierMetrics
	metricSourceProvider *metricSource.SourceProvider
	imageStores          map[string]moira.ImageStore
}

// NewNotifier is initializer for StandardNotifier.
func NewNotifier(database moira.Database, logger moira.Logger, config Config, metrics *metrics.NotifierMetrics, metricSourceProvider *metricSource.SourceProvider, imageStoreMap map[string]moira.ImageStore) *StandardNotifier {
	return &StandardNotifier{
		senders:              make(map[string]chan NotificationPackage),
		logger:               logger,
		database:             database,
		scheduler:            NewScheduler(database, logger, metrics),
		config:               config,
		metrics:              metrics,
		metricSourceProvider: metricSourceProvider,
		imageStores:          imageStoreMap,
	}
}

// Send is realization of StandardNotifier Send functionality.
func (notifier *StandardNotifier) Send(pkg *NotificationPackage, waitGroup *sync.WaitGroup) {
	ch, found := notifier.senders[pkg.Contact.Type]
	if !found {
		notifier.reschedule(pkg, fmt.Sprintf("Unknown sender contact type '%s' [%s]", pkg.Contact.Type, pkg))
		return
	}
	waitGroup.Add(1)
	go func(pkg *NotificationPackage) {
		defer waitGroup.Done()
		getLogWithPackageContext(&notifier.logger, pkg, &notifier.config).
			Debug().
			Interface("package", pkg).
			Msg("Start sending")

		select {
		case ch <- *pkg:
			break
		case <-time.After(notifier.config.SendingTimeout):
			notifier.reschedule(pkg, fmt.Sprintf("Timeout sending %s", pkg))
			break
		}
	}(pkg)
}

// GetSenders get hash of registered notifier senders.
func (notifier *StandardNotifier) GetSenders() map[string]bool {
	hash := make(map[string]bool)
	for key := range notifier.senders {
		hash[key] = true
	}
	return hash
}

// GetReadBatchSize returns amount of messages notifier reads from Redis per iteration.
func (notifier *StandardNotifier) GetReadBatchSize() int64 {
	return notifier.config.ReadBatchSize
}

func (notifier *StandardNotifier) reschedule(pkg *NotificationPackage, reason string) {
	if pkg.DontResend {
		notifier.metrics.MarkSendersDroppedNotifications(pkg.Contact.Type)
		return
	}

	notifier.metrics.MarkSendingFailed()
	notifier.metrics.MarkSendersFailedMetrics(pkg.Contact.Type)

	logger := getLogWithPackageContext(&notifier.logger, pkg, &notifier.config)

	if notifier.needToStop(pkg.FailCount) {
		notifier.metrics.MarkSendersDroppedNotifications(pkg.Contact.Type)
		logger.Error().
			Msg("Stop resending. Notification interval is timed out")
		return
	}

	logger.Warning().
		Int("number_of_retries", pkg.FailCount).
		String("reason", reason).
		Msg("Can't send message. Retry again in 1 min")

	for _, event := range pkg.Events {
		subID := moira.UseString(event.SubscriptionID)
		eventLogger := logger.Clone().String(moira.LogFieldNameSubscriptionID, subID)
		SetLogLevelByConfig(notifier.config.LogSubscriptionsToLevel, subID, &eventLogger)
		params := moira.SchedulerParams{
			Now:               time.Now(),
			Event:             event,
			Trigger:           pkg.Trigger,
			Contact:           pkg.Contact,
			Plotting:          pkg.Plotting,
			ThrottledOld:      pkg.Throttled,
			SendFail:          pkg.FailCount + 1,
			ReschedulingDelay: notifier.config.ReschedulingDelay,
		}
		notification := notifier.scheduler.ScheduleNotification(params, eventLogger)
		if err := notifier.database.AddNotification(notification); err != nil {
			eventLogger.Error().
				Error(err).
				Msg("Failed to save scheduled notification")
		}
	}
}

func (notifier *StandardNotifier) runSender(sender moira.Sender, ch chan NotificationPackage) {
	defer func() {
		if err := recover(); err != nil {
			notifier.logger.Error().
				String(moira.LogFieldNameStackTrace, string(debug.Stack())).
				Interface("recovered_err", err).
				Msg("Notifier panicked")
		}
	}()
	defer notifier.waitGroup.Done()

	for pkg := range ch {
		log := getLogWithPackageContext(&notifier.logger, &pkg, &notifier.config)
		plottingLog := log.Clone().String(moira.LogFieldNameContext, "plotting")
		plots, err := notifier.buildNotificationPackagePlots(pkg, plottingLog)
		if err != nil {
			var event logging.EventBuilder
			switch err.(type) { // nolint:errorlint
			case plotting.ErrNoPointsToRender:
				event = plottingLog.Debug()
			default:
				event = plottingLog.Error()
			}
			event.
				String(moira.LogFieldNameTriggerID, pkg.Trigger.ID).
				Error(err).
				Msg("Can't build notification package plot for trigger")
		}

		err = pkg.Trigger.PopulatedDescription(pkg.Events)
		if err != nil {
			log.Warning().
				Error(err).
				Msg("Error populate description")
		}

		err = sender.SendEvents(pkg.Events, pkg.Contact, pkg.Trigger, plots, pkg.Throttled)
		if err == nil {
			notifier.metrics.MarkSendersOkMetrics(pkg.Contact.Type)
			continue
		}
		switch e := err.(type) { // nolint:errorlint
		case moira.SenderBrokenContactError:
			log.Warning().
				Error(e).
				Msg("Cannot send to broken contact")
			notifier.metrics.MarkSendersDroppedNotifications(pkg.Contact.Type)
		default:
			if pkg.FailCount > notifier.config.MaxFailAttemptToSendAvailable {
				log.Error().
					Error(err).
					Int("fail_count", pkg.FailCount).
					Msg("Cannot send notification")
			} else {
				log.Warning().
					Error(err).
					Msg("Cannot send notification")
			}

			notifier.reschedule(&pkg, err.Error())
		}
	}
}

func (notifier *StandardNotifier) needToStop(failCount int) bool {
	return time.Duration(failCount)*notifier.config.ReschedulingDelay > notifier.config.ResendingTimeout
}
