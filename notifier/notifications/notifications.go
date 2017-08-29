package notifications

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/notifier"
)

//FetchNotificationsWorker - check for new notifications and send it using notifier
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
				worker.Logger.Info("Fetching scheduled notifications stopped")
				worker.Notifier.StopSenders()
				return nil
			case <-checkTicker.C:
				if err := worker.processScheduledNotifications(); err != nil {
					worker.Logger.Warningf("Failed to fetch scheduled notifications: %s", err.Error())
				}
			}
		}
	})
	worker.Logger.Info("Fetching scheduled notifications started")
}

// Stop stops new notifications fetching and wait for finish
func (worker *FetchNotificationsWorker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}

func (worker *FetchNotificationsWorker) processScheduledNotifications() error {
	ts := time.Now()
	notifications, err := worker.Database.GetNotificationsAndDelete(ts.Unix())
	if err != nil {
		return err
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
	}
	var sendingWG sync.WaitGroup
	for _, pkg := range notificationPackages {
		worker.Notifier.Send(pkg, &sendingWG)
	}
	sendingWG.Wait()
	return nil
}
