package notifications

import (
	"fmt"
	"sync"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira/internal/notifier"
)

const sleepAfterNotifierBadState = time.Second * 10

// FetchNotificationsWorker - check for new notifications and send it using notifier
type FetchNotificationsWorker struct {
	Logger   moira2.Logger
	Database moira2.Database
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
						worker.Logger.Warningf("Stop sending notifications for %v: %s. Fix SelfState errors and turn on notifier in /notifications page", sleepAfterNotifierBadState, err.Error())
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
	state, err := worker.Database.GetNotifierState()
	if err != nil {
		return notifierInBadStateError("can't get current notifier state")
	}
	if state != moira2.SelfStateOK {
		return notifierInBadStateError(fmt.Sprintf("notifier in a bad state: %v", state))
	}
	notifications, err := worker.Database.FetchNotifications(time.Now().Unix())
	if err != nil {
		return err
	}
	notificationPackages := make(map[string]*notifier.NotificationPackage)
	for _, notification := range notifications {
		packageKey := fmt.Sprintf("%s:%s:%s", notification.Contact.Type, notification.Contact.Value, notification.Event.TriggerID)
		p, found := notificationPackages[packageKey]
		if !found {
			p = &notifier.NotificationPackage{
				Events:    make([]moira2.NotificationEvent, 0, len(notifications)),
				Trigger:   notification.Trigger,
				Contact:   notification.Contact,
				Plotting:  notification.Plotting,
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
