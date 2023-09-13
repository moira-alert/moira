package selfstate

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
)

func (selfCheck *SelfCheckWorker) selfStateChecker(stop <-chan struct{}) error {
	selfCheck.Logger.Info().Msg("Moira Notifier Self State Monitor started")

	checkTicker := time.NewTicker(defaultCheckInterval)
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

func (selfCheck *SelfCheckWorker) handleCheckServices(nowTS int64) []moira.NotificationEvent {
	var events []moira.NotificationEvent

	for _, heartbeat := range selfCheck.heartbeats {
		currentValue, hasErrors, err := heartbeat.Check(nowTS)
		if err != nil {
			selfCheck.Logger.Error().
				Error(err).
				Msg("Heartbeat failed")
		}

		if hasErrors {
			events = append(events, generateNotificationEvent(heartbeat.GetErrorMessage(), currentValue))
			if heartbeat.NeedTurnOffNotifier() {
				selfCheck.setNotifierState(moira.SelfStateERROR)
			}

			if !heartbeat.NeedToCheckOthers() {
				break
			}
		}
	}

	return events
}

func (selfCheck *SelfCheckWorker) sendNotification(events []moira.NotificationEvent, nowTS int64) int64 {
	eventsJSON, _ := json.Marshal(events)
	selfCheck.Logger.Error().
		Int("number_of_events", len(events)).
		String("events_json", string(eventsJSON)).
		Msg("Health check. Send package notification events")
	selfCheck.sendErrorMessages(events)
	return nowTS + selfCheck.Config.NoticeIntervalSeconds
}

func (selfCheck *SelfCheckWorker) check(nowTS int64, nextSendErrorMessage int64) int64 {
	if nextSendErrorMessage < nowTS {
		events := selfCheck.handleCheckServices(nowTS)
		if len(events) > 0 {
			nextSendErrorMessage = selfCheck.sendNotification(events, nowTS)
		}
	}

	return nextSendErrorMessage
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(events []moira.NotificationEvent) {
	var sendingWG sync.WaitGroup

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

		selfCheck.Notifier.Send(&pkg, &sendingWG)
		sendingWG.Wait()
	}
}

func generateNotificationEvent(message string, currentValue int64) moira.NotificationEvent {
	val := float64(currentValue)
	return moira.NotificationEvent{
		Timestamp: time.Now().Unix(),
		OldState:  moira.StateNODATA,
		State:     moira.StateERROR,
		Metric:    message,
		Value:     &val,
	}
}

func (selfCheck *SelfCheckWorker) setNotifierState(state string) {
	err := selfCheck.Database.SetNotifierState(state)
	if err != nil {
		selfCheck.Logger.Error().
			Error(err).
			Msg("Can't set notifier state")
	}
}
