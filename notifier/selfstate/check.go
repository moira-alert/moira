package selfstate

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

type heartbeatNotificationEvent struct {
	moira.NotificationEvent
	heartbeat.CheckTags
}

func (selfCheck *SelfCheckWorker) selfStateChecker(stop <-chan struct{}) error {
	selfCheck.Logger.Info().Msg("Moira Notifier Self State Monitor started")

	checkTicker := time.NewTicker(selfCheck.Config.CheckInterval)
	defer checkTicker.Stop()

	nextSendErrorMessage := time.Now().Unix()

	for {
		select {
		case <-stop:
			selfCheck.Logger.Info().Msg("Moira Notifier Self State Monitor stopped")
			return nil
		case <-checkTicker.C:
			selfCheck.Logger.Debug().
				Int64("nextSendErrorMessage", nextSendErrorMessage).
				Msg("call check")

			nextSendErrorMessage = selfCheck.check(time.Now().Unix(), nextSendErrorMessage)
		}
	}
}

func (selfCheck *SelfCheckWorker) handleCheckServices(nowTS int64) []heartbeatNotificationEvent {
	var events []heartbeatNotificationEvent

	checksGraph := constructHeartbeatsGraph(selfCheck.heartbeats)
	checksResult, err := checksGraph.executeGraph(nowTS)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Heartbeats failed")
	}

	if checksResult.hasErrors {
		errorMessage := strings.Join(checksResult.errorMessages, "\n")
		events = append(events, heartbeatNotificationEvent{
			NotificationEvent: generateNotificationEvent(errorMessage, checksResult.lastSuccessCheckElapsedTime, nowTS, moira.StateNODATA, moira.StateERROR),
			CheckTags:         checksResult.checksTags,
		})

		if checksResult.needTurnOffNotifier {
			selfCheck.setNotifierState(moira.SelfStateERROR, checksResult.checksTags)
		}

	} else {

		toNotifyCheckTags, notifierStateChanged, err := selfCheck.enableNotifierIfNeed()
		if err != nil {
			selfCheck.Logger.Error().
				Error(err).
				Msg("Enabling notifier failed")
		} else if notifierStateChanged {
			events = append(events, heartbeatNotificationEvent{
				NotificationEvent: generateNotificationEvent("Moira notifications enabled", 0, nowTS, moira.StateERROR, moira.StateOK),
				CheckTags:         toNotifyCheckTags,
			})
		}
	}

	return events
}

func (selfCheck *SelfCheckWorker) sendNotification(events []heartbeatNotificationEvent, nowTS int64) int64 {
	eventsJSON, _ := json.Marshal(events)
	selfCheck.Logger.Error().
		Int("number_of_events", len(events)).
		String("events_json", string(eventsJSON)).
		Msg("Health check. Send package notification events")
	selfCheck.sendMessages(events)
	return nowTS + selfCheck.Config.NoticeIntervalSeconds
}

func (selfCheck *SelfCheckWorker) check(nowTS int64, nextSendErrorMessage int64) int64 {
	events := selfCheck.handleCheckServices(nowTS)
	if nextSendErrorMessage < nowTS && len(events) > 0 {
		nextSendErrorMessage = selfCheck.sendNotification(events, nowTS)
	}

	return nextSendErrorMessage
}

func (selfCheck *SelfCheckWorker) constructUserNotification(events []heartbeatNotificationEvent) ([]*notifier.NotificationPackage, error) {
	contactToEvents := make(map[*moira.ContactData][]moira.NotificationEvent)
	for _, event := range events {
		if len(event.CheckTags) == 0 {
			continue
		}

		subscriptions, err := selfCheck.Database.GetTagsSubscriptions(event.CheckTags)
		if err != nil {
			return nil, err
		}

		for _, subscription := range subscriptions {
			contacts, err := selfCheck.Database.GetContacts(subscription.Contacts)
			if err != nil {
				return nil, err
			}
			for _, contact := range contacts {
				contactToEvents[contact] = append(contactToEvents[contact], event.NotificationEvent)
			}
		}
	}

	notificationPkgs := make([]*notifier.NotificationPackage, 0, len(contactToEvents))
	for contact, events := range contactToEvents {
		notificationPkgs = append(notificationPkgs, &notifier.NotificationPackage{
			Contact: *contact,
			Trigger: moira.TriggerData{
				Name:       "Moira health check",
				ErrorValue: float64(0),
			},
			Events:     events,
			DontResend: true,
		})
	}

	return notificationPkgs, nil
}

func (selfCheck *SelfCheckWorker) sendMessages(events []heartbeatNotificationEvent) {
	var sendingWG sync.WaitGroup

	selfCheck.sendNotificationToAdmins(moira.Map(
		events,
		func(et heartbeatNotificationEvent) moira.NotificationEvent { return et.NotificationEvent },
	),
		&sendingWG,
	)
	sendingWG.Wait()

	selfCheck.sendNotificationToUsers(events, &sendingWG)
	sendingWG.Wait()
}

func (selfCheck *SelfCheckWorker) sendNotificationToUsers(events []heartbeatNotificationEvent, sendingWG *sync.WaitGroup) {
	notificationPackages, err := selfCheck.constructUserNotification(events)
	if err != nil {
		selfCheck.Logger.Warning().
			Error(err).
			Msg("Sending notifications via subscriptions has failed")
	}

	for _, pkg := range notificationPackages {
		if pkg == nil {
			continue
		}

		selfCheck.Notifier.Send(pkg, sendingWG)
	}
}

func (selfCheck *SelfCheckWorker) sendNotificationToAdmins(events []moira.NotificationEvent, sendingWG *sync.WaitGroup) {
	for _, adminContact := range selfCheck.Config.Contacts {
		pkg := notifier.NotificationPackage{
			Contact: moira.ContactData{
				Type:  adminContact["type"],
				Value: adminContact["value"],
			},
			Trigger: moira.TriggerData{
				Name:       "Moira health check",
				ErrorValue: float64(0),
			},
			Events:     events,
			DontResend: true,
		}

		selfCheck.Notifier.Send(&pkg, sendingWG)
	}
}

func generateNotificationEvent(message string, lastSuccessCheckElapsedTime, timestamp int64, oldState, state moira.State) moira.NotificationEvent {
	val := float64(lastSuccessCheckElapsedTime)
	return moira.NotificationEvent{
		Timestamp: timestamp,
		OldState:  oldState,
		State:     state,
		Metric:    message,
		Value:     &val,
	}
}

func (selfCheck *SelfCheckWorker) enableNotifierIfNeed() ([]string, bool, error) {
	notifierState, err := selfCheck.Database.GetNotifierState()
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't get actual notifier state")
		return notifierState.ToNotifyTags, false, err
	}

	if notifierState.NewState == moira.SelfStateOK {
		selfCheck.Logger.Info().
			Msg("Can't enable notifier: notifier is already enabled")
		return notifierState.ToNotifyTags, false, nil
	}

	if notifierState.NewState == moira.SelfStateERROR && notifierState.Actor == moira.SelfStateActorManual {
		selfCheck.Logger.Warning().
			Msg("Can't enable notifier: notifier is disabled manually")
		return notifierState.ToNotifyTags, false, nil
	}

	if err = selfCheck.setNotifierState(moira.SelfStateOK, notifierState.ToNotifyTags); err == nil {
		return notifierState.ToNotifyTags, true, err
	}

	return notifierState.ToNotifyTags, false, nil
}

func (selfCheck *SelfCheckWorker) setNotifierState(state string, checksTags heartbeat.CheckTags) error {
	err := selfCheck.Database.SetNotifierState(moira.SelfStateActorAutomatic, state, checksTags)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't set notifier state")
	}
	return err
}
