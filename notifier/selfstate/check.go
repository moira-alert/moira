package selfstate

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

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

type NotificationEventAndCheckTags struct{moira.NotificationEvent; heartbeat.CheckTags}

func (selfCheck *SelfCheckWorker) handleCheckServices(nowTS int64) []NotificationEventAndCheckTags {
	var events []NotificationEventAndCheckTags

	for _, heartbeat := range selfCheck.heartbeats {
		currentValue, hasErrors, err := heartbeat.Check(nowTS)
		if err != nil {
			selfCheck.Logger.Error().
				Error(err).
				Msg("Heartbeat failed")
		}

		if hasErrors {
			events = append(events,	NotificationEventAndCheckTags{
				NotificationEvent: generateNotificationEvent(heartbeat.GetErrorMessage(), currentValue),
				CheckTags: heartbeat.GetCheckTags(),
			})
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

func (selfCheck *SelfCheckWorker) sendNotification(events []NotificationEventAndCheckTags, nowTS int64) int64 {
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

func (selfCheck *SelfCheckWorker) eventsToContacts(events []NotificationEventAndCheckTags) ([]struct{*moira.ContactData; *moira.NotificationEvents}, error) {
	result := make(map[*moira.ContactData][]moira.NotificationEvent)
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
				result[contact] = append(result[contact], event.NotificationEvent)
			}
		}
	}

	var resultList []struct{*moira.ContactData; *moira.NotificationEvents}
	for contact, events := range result {
		r := moira.NotificationEvents(events)
		resultList = append(resultList, struct{*moira.ContactData; *moira.NotificationEvents}{contact, &r})
	}

	return resultList, nil
}

func (selfCheck *SelfCheckWorker) sendErrorMessages(events []NotificationEventAndCheckTags) {
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
			Events:     moira.Map(events, func(et NotificationEventAndCheckTags) moira.NotificationEvent { return et.NotificationEvent }),
			DontResend: true,
		}

		selfCheck.Notifier.Send(&pkg, &sendingWG)
		sendingWG.Wait()
	}

	eventsAndContacts, err := selfCheck.eventsToContacts(events)
	if err != nil {
		selfCheck.Logger.Warning().
		Error(err).
		Msg("Sending notifications via subscriptions has failed")
	}

	for _, contactAndEvent := range eventsAndContacts {
		pkg := notifier.NotificationPackage{
			Contact: *contactAndEvent.ContactData,
			Trigger: moira.TriggerData{
				Name: "Moira health check",
				ErrorValue: float64(0),
			},
			Events: *contactAndEvent.NotificationEvents,
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
