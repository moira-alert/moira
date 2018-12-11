package notifier

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/metrics/graphite"
	"github.com/moira-alert/moira/plotting"
	"github.com/moira-alert/moira/remote"
	"github.com/moira-alert/moira/target"
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

func (pkg NotificationPackage) String() string {
	return fmt.Sprintf("package of %d notifications to %s", len(pkg.Events), pkg.Contact.Value)
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
	waitGroup sync.WaitGroup
	senders   map[string]chan NotificationPackage
	logger    moira.Logger
	database  moira.Database
	scheduler Scheduler
	config    Config
	metrics   *graphite.NotifierMetrics
}

// NewNotifier is initializer for StandardNotifier
func NewNotifier(database moira.Database, logger moira.Logger, config Config, metrics *graphite.NotifierMetrics) *StandardNotifier {
	return &StandardNotifier{
		senders:   make(map[string]chan NotificationPackage),
		logger:    logger,
		database:  database,
		scheduler: NewScheduler(database, logger, metrics),
		config:    config,
		metrics:   metrics,
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
	if metric, found := notifier.metrics.SendersFailedMetrics.GetMetric(pkg.Contact.Type); found {
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

func (notifier *StandardNotifier) run(sender moira.Sender, ch chan NotificationPackage) {
	defer notifier.waitGroup.Done()
	for pkg := range ch {
		plot, err := notifier.getNotificationPackagePlot(pkg)
		if err != nil {
			notifier.logger.Errorf("Can't get notification package plot for %s: %s", pkg.Trigger, err.Error())
		}
		err = sender.SendEvents(pkg.Events, pkg.Contact, pkg.Trigger, plot, pkg.Throttled)
		if err == nil {
			if metric, found := notifier.metrics.SendersOkMetrics.GetMetric(pkg.Contact.Type); found {
				metric.Mark(1)
			}
		} else {
			notifier.resend(&pkg, err.Error())
		}
	}
}

func (notifier *StandardNotifier) getNotificationPackagePlot(pkg NotificationPackage) ([]byte, error) {

	buff := bytes.NewBuffer(make([]byte, 0))

	if pkg.Trigger.ID == "" {
		return buff.Bytes(), nil
	}

	trigger, err := notifier.database.GetTrigger(pkg.Trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}

	plotTemplate, err := plotting.GetPlotTemplate(pkg.Plotting.Theme)
	if err != nil {
		return buff.Bytes(), err
	}

	remoteCfg := &remote.Config{Enabled: false}

	to := time.Now().UTC()
	from := to.Add(-60 * time.Minute)

	tts, err := getTriggerEvaluationResult(notifier.database, remoteCfg, from.Unix(), to.Unix(), trigger.ID)
	if err != nil {
		return buff.Bytes(), err
	}

	var metricsData = make([]*types.MetricData, 0, len(tts.Main)+len(tts.Additional))
	for _, ts := range tts.Main {
		metricsData = append(metricsData, &ts.MetricData)
	}
	for _, ts := range tts.Additional {
		metricsData = append(metricsData, &ts.MetricData)
	}

	metricsToShow := make([]string, 0)

	for _, event := range pkg.Events {
		metricsToShow = append(metricsToShow, event.Metric)
	}

	renderable := plotTemplate.GetRenderable(&trigger, metricsData, metricsToShow)

	notifier.logger.Debugf("Attempt to render %s timeseries: %s", trigger.ID,
		strings.Join(metricsToShow, ", "))

	if err = renderable.Render(chart.PNG, buff); err != nil {
		return buff.Bytes(), err
	}

	return buff.Bytes(), nil
}

func getTriggerEvaluationResult(dataBase moira.Database, remoteConfig *remote.Config,
	from, to int64, triggerID string) (*checker.TriggerTimeSeries, error) {
	allowRealtimeAllerting := true
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return nil, err
	}
	triggerMetrics := &checker.TriggerTimeSeries{
		Main:       make([]*target.TimeSeries, 0),
		Additional: make([]*target.TimeSeries, 0),
	}
	if trigger.IsRemote && !remoteConfig.IsEnabled() {
		return nil, remote.ErrRemoteStorageDisabled
	}
	for i, tar := range trigger.Targets {
		var timeSeries []*target.TimeSeries
		if trigger.IsRemote {
			timeSeries, err = remote.Fetch(remoteConfig, tar, from, to, allowRealtimeAllerting)
			if err != nil {
				return nil, err
			}
		} else {
			result, err := target.EvaluateTarget(dataBase, tar, from, to, allowRealtimeAllerting)
			if err != nil {
				return nil, err
			}
			timeSeries = result.TimeSeries
		}
		if i == 0 {
			triggerMetrics.Main = timeSeries
		} else {
			triggerMetrics.Additional = append(triggerMetrics.Additional, timeSeries...)
		}
	}
	return triggerMetrics, nil
}
