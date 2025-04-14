package selfstate

import (
	"encoding/json"
	"fmt"
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
	checksResult, err := selfCheck.heartbeatsGraph.executeGraph(nowTS)
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
		if selfCheck.state != moira.SelfStateWorkerERROR {
			selfCheck.updateState(moira.SelfStateWorkerWARN)
		}

		if graphResult.needTurnOffNotifier {
			if err := selfCheck.setNotifierState(moira.SelfStateERROR); err != nil {
				selfCheck.Logger.Error().
					Error(err).
					Msg("Disabling notifier failed")
			}
		}

		if selfCheck.hasStateChanged() {
			errorMessage := strings.Join(graphResult.errorMessages, "\n")
			events = append(events, heartbeatNotificationEvent{
				NotificationEvent: generateNotificationEvent(errorMessage, graphResult.lastSuccessCheckElapsedTime, nowTS, moira.StateNODATA, moira.StateERROR),
				CheckTags:         graphResult.checksTags,
			})
		}
	} else {
		selfCheck.updateState(moira.SelfStateWorkerOK)
		selfCheck.lastSuccessChecksResult = graphResult
		notifierEnabled, err := selfCheck.enableNotifierIfPossible()

		if err != nil {
			selfCheck.Logger.Error().
				Error(err).
				Msg("Enabling notifier failed")
		} else if notifierEnabled {
			events = append(events, heartbeatNotificationEvent{
				NotificationEvent: generateNotificationEvent("Moira notifications enabled", 0, nowTS, moira.StateERROR, moira.StateOK),
				CheckTags:         selfCheck.lastChecksResult.checksTags,
			})
		}
	}

	if graphResult.nowTimestamp-selfCheck.lastSuccessChecksResult.nowTimestamp > selfCheck.Config.UserNotificationsInterval {
		selfCheck.updateState(moira.SelfStateWorkerERROR)
	}

	selfCheck.lastChecksResult = graphResult

	return events
}

func (selfCheck *SelfCheckWorker) updateState(newState moira.SelfStateWorkerState) {
	selfCheck.oldState = selfCheck.state
	selfCheck.state = newState
}

func (selfCheck *SelfCheckWorker) hasStateChanged() bool {
	return selfCheck.state != selfCheck.oldState
}

func (selfCheck *SelfCheckWorker) shouldNotifyUsers() bool {
	return selfCheck.oldState == moira.SelfStateWorkerWARN && selfCheck.state == moira.SelfStateWorkerERROR ||
		selfCheck.oldState == moira.SelfStateWorkerERROR && selfCheck.state == moira.SelfStateWorkerOK
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
	selfCheck.Logger.Info().
		Msg(fmt.Sprintf("Handle check services events count: %v", len(events)))
	selfCheck.Logger.Info().
		Msg(fmt.Sprintf("nextSendErrorMessage < nowTS: %v", nextSendErrorMessage < nowTS))
	// if nextSendErrorMessage < nowTS && len(events) > 0 {
	if len(events) > 0 {
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

	if selfCheck.shouldNotifyUsers() {
		selfCheck.sendNotificationToUsers(events, &sendingWG)
	}

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

func (selfCheck *SelfCheckWorker) enableNotifierIfPossible() (bool, error) {
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
