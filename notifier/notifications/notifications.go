package notifications

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
)

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
					worker.Logger.Warningf("Failed to fetch scheduled notifications: %s", err.Error())
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
		worker.Logger.Error("can't get current notifier state")
		return fmt.Errorf("can't get current notifier state")
	}
	if state != "OK" {
		worker.Logger.Errorf("Stop sending notifications. Current notifier state: %v", state)
		return fmt.Errorf("stop sending notifications. Current notifier state: %v", state)
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
