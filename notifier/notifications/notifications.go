package notifications

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/wcharczuk/go-chart"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/plotting"
)

const sleepAfterNotifierBadState = time.Second * 10

// FetchNotificationsWorker - check for new notifications and send it using notifier
type FetchNotificationsWorker struct {
	Logger   moira.Logger
	Database moira.Database
	Notifier notifier.Notifier
	tomb     tomb.Tomb
}

// Start is a cycle that fetches scheduled notifications from database
func (worker *FetchNotificationsWorker) Start() {
	worker.tomb.Go(func() error {
		checkTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-worker.tomb.Dying():
				worker.Logger.Info("Moira Notifier Fetching scheduled notifications stopped")
				worker.Notifier.StopSenders()
				return nil
			case <-checkTicker.C:
				if err := worker.processScheduledNotifications(); err != nil {
					switch err.(type) {
					case notifierInBadStateError:
						worker.Logger.Warningf("Stop sending notifications for %v: %s", sleepAfterNotifierBadState, err.Error())
						<-time.After(sleepAfterNotifierBadState)
					default:
						worker.Logger.Warningf("Failed to fetch scheduled notifications: %s", err.Error())
					}
				}
			}
		}
	})
	worker.Logger.Info("Moira Notifier Fetching scheduled notifications started")
}

// Stop stops new notifications fetching and wait for finish
func (worker *FetchNotificationsWorker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}

func (worker *FetchNotificationsWorker) processScheduledNotifications() error {
	notifications, err := worker.Database.FetchNotifications(time.Now().Unix())
	if err != nil {
		return err
	}
	state, err := worker.Database.GetNotifierState()
	if err != nil {
		return notifierInBadStateError("can't get current notifier state")
	}
	if state != "OK" {
		return notifierInBadStateError(fmt.Sprintf("notifier in a bad state: %v", state))
	}

	notificationPackages := make(map[string]*notifier.NotificationPackage)
	for _, notification := range notifications {
		packageKey := fmt.Sprintf("%s:%s:%s", notification.Contact.Type, notification.Contact.Value, notification.Event.TriggerID)
		p, found := notificationPackages[packageKey]
		if !found {
			p = &notifier.NotificationPackage{
				Events:    make([]moira.NotificationEvent, 0, len(notifications)),
				Trigger:   notification.Trigger,
				Contact:   notification.Contact,
				Throttled: notification.Throttled,
				FailCount: notification.SendFail,
			}
		}
		p.Events = append(p.Events, notification.Event)
		notificationPackages[packageKey] = p

		if notification.Plotting.Enabled {
			plot, err := worker.getNotificationPackagePlot(notification.Trigger, p.Events, notification.Plotting.Theme)
			if err != nil {
				p.Plot = plot
			}
		}

	}
	var sendingWG sync.WaitGroup
	for _, pkg := range notificationPackages {
		worker.Notifier.Send(pkg, &sendingWG)
	}
	sendingWG.Wait()
	return nil
}

func (worker *FetchNotificationsWorker) getNotificationPackagePlot(triggerData moira.TriggerData,
	events []moira.NotificationEvent, plotTheme string) ([]byte, error) {

	buff := bytes.NewBuffer(make([]byte, 0))

	trigger, err := worker.Database.GetTrigger(triggerData.ID)
	if err != nil {
		return nil, err
	}

	plotTemplate, err := plotting.GetPlotTemplate(plotTheme)
	if err != nil {
		return nil, err
	}

	var metricsData []*types.MetricData

	metricsToShow := make([]string, 0)

	for _, event := range events {
		metricsToShow = append(metricsToShow, event.Metric)
	}

	renderable := plotTemplate.GetRenderable(&trigger, metricsData, metricsToShow)
	if err = renderable.Render(chart.PNG, buff); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
