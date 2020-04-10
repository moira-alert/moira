package notifier

import (
	"fmt"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/plotting"
)

// NotificationPackage represent sending data
type NotificationPackage struct {
	Events     []moira.NotificationEvent
	Trigger    moira.TriggerData
	Contact    moira.ContactData
	Plotting   moira.PlottingData
	FailCount  int
	Throttled  bool
	DontResend bool
}

// String returns notification package summary
func (pkg NotificationPackage) String() string {
	return fmt.Sprintf("package of %d notifications to %s", len(pkg.Events), pkg.Contact.Value)
}

// GetWindow returns the earliest and the latest notification package timestamps
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

// GetMetricNames returns all metric names found in package events
func (pkg NotificationPackage) GetMetricNames() []string {
	metricNames := make([]string, 0)
	for _, event := range pkg.Events {
		if !event.IsTriggerEvent {
			metricNames = append(metricNames, event.Metric)
		}
	}
	return metricNames
}

// Notifier implements notification functionality
type Notifier interface {
	Send(pkg *NotificationPackage, waitGroup *sync.WaitGroup)
	RegisterSender(senderSettings map[string]string, sender moira.Sender) error
	StopSenders()
	GetSenders() map[string]bool
}

// StandardNotifier represent notification functionality
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

// NewNotifier is initializer for StandardNotifier
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

// Send is realization of StandardNotifier Send functionality
func (notifier *StandardNotifier) Send(pkg *NotificationPackage, waitGroup *sync.WaitGroup) {
	ch, found := notifier.senders[pkg.Contact.Type]
	if !found {
		notifier.resend(pkg, fmt.Sprintf("Unknown contact type '%s' [%s]", pkg.Contact.Type, pkg))
		return
	}
	waitGroup.Add(1)
	go func(pkg *NotificationPackage) {
		defer waitGroup.Done()
		notifier.logger.Debugf("Start sending %s", pkg)
		select {
		case ch <- *pkg:
			break
		case <-time.After(notifier.config.SendingTimeout):
			notifier.resend(pkg, fmt.Sprintf("Timeout sending %s", pkg))
			break
		}
	}(pkg)
}

// GetSenders get hash of registered notifier senders
func (notifier *StandardNotifier) GetSenders() map[string]bool {
	hash := make(map[string]bool)
	for key := range notifier.senders {
		hash[key] = true
	}
	return hash
}

func (notifier *StandardNotifier) resend(pkg *NotificationPackage, reason string) {
	if pkg.DontResend {
		return
	}
	notifier.metrics.SendingFailed.Mark(1)
	if metric, found := notifier.metrics.SendersFailedMetrics.GetRegisteredMeter(pkg.Contact.Type); found {
		metric.Mark(1)
	}
	notifier.logger.Warningf("Can't send message after %d try: %s. Retry again after 1 min", pkg.FailCount, reason)
	if time.Duration(pkg.FailCount)*time.Minute > notifier.config.ResendingTimeout {
		notifier.logger.Error("Stop resending. Notification interval is timed out")
	} else {
		for _, event := range pkg.Events {
			notification := notifier.scheduler.ScheduleNotification(time.Now(), event,
				pkg.Trigger, pkg.Contact, pkg.Plotting, pkg.Throttled, pkg.FailCount+1)
			if err := notifier.database.AddNotification(notification); err != nil {
				notifier.logger.Errorf("Failed to save scheduled notification: %s", err)
			}
		}
	}
}

func (notifier *StandardNotifier) runSender(sender moira.Sender, ch chan NotificationPackage) {
	defer notifier.waitGroup.Done()
	for pkg := range ch {
		plot, err := notifier.buildNotificationPackagePlot(pkg)
		if err != nil {
			buildErr := fmt.Sprintf("Can't build notification package plot for %s: %s", pkg.Trigger.ID, err.Error())
			switch err.(type) {
			case plotting.ErrNoPointsToRender:
				notifier.logger.Debugf(buildErr)
			default:
				notifier.logger.Errorf(buildErr)
			}
		}

		pkg.Trigger.Desc, err = pkg.Trigger.GetPopulatedDescription(pkg.Events)
		if err != nil {
			notifier.logger.Errorf("Error populate description: %v", err)
		}

		err = sender.SendEvents(pkg.Events, pkg.Contact, pkg.Trigger, plot, pkg.Throttled)
		if err == nil {
			if metric, found := notifier.metrics.SendersOkMetrics.GetRegisteredMeter(pkg.Contact.Type); found {
				metric.Mark(1)
			}
		} else {
			notifier.resend(&pkg, err.Error())
		}
	}
}
