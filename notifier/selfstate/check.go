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
	checksGraph := constructHeartbeatsGraph(selfCheck.heartbeats)

	checksResult, err := checksGraph.executeGraph(nowTS)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Heartbeats failed")
	}

	events := selfCheck.handleGraphExecutionResult(nowTS, checksResult)

	return events
}

func (selfCheck *SelfCheckWorker) handleGraphExecutionResult(nowTS int64, graphResult graphExecutionResult) []heartbeatNotificationEvent {
	var events []heartbeatNotificationEvent

	if graphResult.hasErrors {
		if graphResult.needTurnOffNotifier {
			if err := selfCheck.setNotifierState(moira.SelfStateERROR); err != nil {
				selfCheck.Logger.Error().
					Error(err).
					Msg("Disabling notifier failed")
			}
		}

		errorMessage := strings.Join(graphResult.errorMessages, "\n")
		events = append(events, heartbeatNotificationEvent{
			NotificationEvent: generateNotificationEvent(errorMessage, graphResult.lastSuccessCheckElapsedTime, nowTS, moira.StateNODATA, moira.StateERROR),
			CheckTags:         graphResult.checksTags,
		})
	} else {
		notifierEnabled, err := selfCheck.enableNotifierIfCan()

		if err != nil {
			selfCheck.Logger.Error().
				Error(err).
				Msg("Enabling notifier failed")
		} else if notifierEnabled {
			selfCheck.Logger.Info().
				Msg("Notifier enabled automatically")
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

func (selfCheck *SelfCheckWorker) enableNotifierIfCan() (bool, error) {
	currentNotifierState, err := selfCheck.Database.GetNotifierState()
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't get actual notifier state")

		return false, err
	}

	if currentNotifierState.Actor == moira.SelfStateActorAutomatic && currentNotifierState.State == moira.SelfStateERROR ||
		currentNotifierState.Actor == moira.SelfStateActorManual && currentNotifierState.State == moira.SelfStateOK {
		if err = selfCheck.setNotifierState(moira.SelfStateOK); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (selfCheck *SelfCheckWorker) setNotifierState(state string) error {
	err := selfCheck.Database.SetNotifierState(moira.SelfStateActorAutomatic, state)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't set notifier state")
	}

	return err
}
