package notifications

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/notifier"
)

const sleepAfterNotifierBadState = time.Second * 10

// FetchNotificationsWorker - check for new notifications and send it using notifier.
type FetchNotificationsWorker struct {
	Logger   moira.Logger
	Database moira.Database
	Notifier notifier.Notifier
	Metrics  *metrics.NotifierMetrics
	tomb     tomb.Tomb
}

func (worker *FetchNotificationsWorker) updateFetchNotificationsMetric(fetchNotificationsStartTime time.Time) {
	if worker.Metrics == nil {
		worker.Logger.Warning().Msg("Cannot update fetch notifications metric because Metrics is nil")
		return
	}

	worker.Metrics.UpdateFetchNotificationsDurationMs(fetchNotificationsStartTime)
}

// Start is a cycle that fetches scheduled notifications from database.
func (worker *FetchNotificationsWorker) Start() {
	worker.tomb.Go(func() error {
		checkTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-worker.tomb.Dying():
				worker.Logger.Info().Msg("Moira Notifier Fetching scheduled notifications stopped")
				worker.Notifier.StopSenders()
				return nil
			case <-checkTicker.C:
				if err := worker.processScheduledNotifications(); err != nil {
					switch err.(type) { // nolint:errorlint
					case notifierInBadStateError:
						worker.Logger.Warning().
							String("stop_sending_notifications_for", sleepAfterNotifierBadState.String()).
							Error(err).
							Msg("Stop sending notifications for some time. Fix SelfState errors and turn on notifier in /notifications page")
						<-time.After(sleepAfterNotifierBadState)
					default:
						worker.Logger.Warning().
							Error(err).
							Msg("Failed to fetch scheduled notifications")
					}
				}
			}
		}
	})
	worker.Logger.Info().Msg("Moira Notifier Fetching scheduled notifications started")
}

// Stop stops new notifications fetching and wait for finish.
func (worker *FetchNotificationsWorker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}

func (worker *FetchNotificationsWorker) processScheduledNotifications() error {
	state, err := worker.Database.GetNotifierState()
	if err != nil {
		return notifierInBadStateError("can't get current notifier state")
	}

	if state != moira.SelfStateOK {
		return notifierInBadStateError(fmt.Sprintf("notifier in a bad state: %v", state))
	}

	fetchNotificationsStartTime := time.Now()
	notifications, err := worker.Database.FetchNotifications(time.Now().Unix(), worker.Notifier.GetReadBatchSize())
	if err != nil {
		return err
	}
	worker.updateFetchNotificationsMetric(fetchNotificationsStartTime)

	notificationPackages := make(map[string]*notifier.NotificationPackage)
	for _, notification := range notifications {
		packageKey := fmt.Sprintf("%s:%s:%s", notification.Contact.Type, notification.Contact.Value, notification.Event.TriggerID)
		p, found := notificationPackages[packageKey]
		if !found {
			p = &notifier.NotificationPackage{
				Events:    make([]moira.NotificationEvent, 0, len(notifications)),
				Trigger:   notification.Trigger,
				Contact:   notification.Contact,
				Plotting:  notification.Plotting,
				Throttled: notification.Throttled,
				FailCount: notification.SendFail,
			}
		}
		p.Events = append(p.Events, notification.Event)

		err = worker.Database.PushContactNotificationToHistory(notification)

		if err != nil {
			worker.Logger.Warning().Error(err).Msg("Can't save notification to history")
		}

		notificationPackages[packageKey] = p
	}
	var sendingWG sync.WaitGroup
	for _, pkg := range notificationPackages {
		worker.Notifier.Send(pkg, &sendingWG)
	}
	sendingWG.Wait()
	return nil
}
