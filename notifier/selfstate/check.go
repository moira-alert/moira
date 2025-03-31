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

	checksGraph := ConstructHeartbeatsGraph(selfCheck.heartbeats)
	checksResult, err := ExecuteGraph(checksGraph, nowTS)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Heartbeats failed")
	}

	if checksResult.hasErrors {
		errorMessage := strings.Join(checksResult.errorMessages, "\n")
		events = append(events, heartbeatNotificationEvent{
			NotificationEvent: generateNotificationEvent(errorMessage, checksResult.currentValue, nowTS),
			CheckTags:         checksResult.checksTags,
		})

		if checksResult.needTurnOffNotifier {
			selfCheck.setNotifierState(moira.SelfStateERROR)
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
	selfCheck.sendErrorMessages(events)
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

func (selfCheck *SelfCheckWorker) sendErrorMessages(events []heartbeatNotificationEvent) {
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

func generateNotificationEvent(message string, currentValue, timestamp int64) moira.NotificationEvent {
	val := float64(currentValue)
	return moira.NotificationEvent{
		Timestamp: timestamp,
		OldState:  moira.StateNODATA,
		State:     moira.StateERROR,
		Metric:    message,
		Value:     &val,
	}
}

func (selfCheck *SelfCheckWorker) setNotifierState(state string) {
	err := selfCheck.Database.SetNotifierState(moira.SelfStateActorAutomatic, state)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't set notifier state")
	}
}
