package notifications

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
	"sync"
	"time"
)

type FetchNotificationsWorker struct {
	logger   moira_alert.Logger
	database moira_alert.Database
	notifier *notifier.Notifier
}

func Init(database moira_alert.Database, logger moira_alert.Logger, sender *notifier.Notifier) FetchNotificationsWorker {
	return FetchNotificationsWorker{
		logger:   logger,
		database: database,
		notifier: sender,
	}
}

// FetchScheduledNotifications is a cycle that fetches scheduled notifications from database
func (worker FetchNotificationsWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	worker.logger.Debug("Start Fetch Sheduled Notifications")
	for {
		select {
		case <-shutdown:
			{
				worker.logger.Debug("Stop Fetch Sheduled Notifications")
				worker.notifier.StopSenders()
				return
			}
		default:
			{
				if err := worker.processScheduledNotifications(); err != nil {
					worker.logger.Warningf("Failed to fetch scheduled notifications: %s", err.Error())
				}
				time.Sleep(time.Second)
			}
		}
	}
}

func (worker *FetchNotificationsWorker) processScheduledNotifications() error {
	ts := time.Now()
	notifications, err := worker.database.GetNotifications(ts.Unix())
	if err != nil {
		return err
	}
	notificationPackages := make(map[string]*notifier.NotificationPackage)
	for _, notification := range notifications {
		packageKey := fmt.Sprintf("%s:%s:%s", notification.Contact.Type, notification.Contact.Value, notification.Event.TriggerID)
		p, found := notificationPackages[packageKey]
		if !found {
			p = &notifier.NotificationPackage{
				Events:    make([]moira_alert.EventData, 0, len(notifications)),
				Trigger:   notification.Trigger,
				Contact:   notification.Contact,
				Throttled: notification.Throttled,
				FailCount: notification.SendFail,
			}
		}
		p.Events = append(p.Events, notification.Event)
		notificationPackages[packageKey] = p
	}
	var sendingWG sync.WaitGroup
	for _, pkg := range notificationPackages {
		worker.notifier.Send(pkg, &sendingWG)
	}
	sendingWG.Wait()
	return nil
}
