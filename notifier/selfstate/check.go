package selfstate

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	NotifyAboutEnabledNotifier bool
}

func (selfCheck *SelfCheckWorker) selfStateChecker(stop <-chan struct{}) error {
	selfCheck.Logger.Info().Msg("Moira Notifier Self State Monitor started")

	checkTicker := time.NewTicker(selfCheck.Config.CheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			selfCheck.Logger.Info().Msg("Moira Notifier Self State Monitor stopped")
			return nil
		case tickTime := <-checkTicker.C:
			selfCheck.Logger.Debug().
				Msg("call check")

			selfCheck.check(tickTime.Unix())
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

		if graphResult.nowTimestamp-selfCheck.lastSuccessChecksResult.nowTimestamp > selfCheck.Config.UserNotificationsInterval {
			selfCheck.updateState(moira.SelfStateWorkerERROR)
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
				NotificationEvent:          generateNotificationEvent(errorMessage, graphResult.lastSuccessCheckElapsedTime, nowTS, moira.StateNODATA, moira.StateERROR),
				CheckTags:                  graphResult.checksTags,
				NotifyAboutEnabledNotifier: false,
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
				NotificationEvent:          generateNotificationEvent("Moira notifications enabled", 0, nowTS, moira.StateERROR, moira.StateOK),
				CheckTags:                  selfCheck.lastChecksResult.checksTags,
				NotifyAboutEnabledNotifier: true,
			})
		}
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

func (selfCheck *SelfCheckWorker) sendNotification(events []heartbeatNotificationEvent) {
	eventsJSON, _ := json.Marshal(events)
	selfCheck.Logger.Error().
		Int("number_of_events", len(events)).
		String("events_json", string(eventsJSON)).
		Msg("Health check. Send package notification events")
	selfCheck.sendMessages(events)
}

func (selfCheck *SelfCheckWorker) check(nowTS int64) {
	events := selfCheck.handleCheckServices(nowTS)
	if len(events) > 0 {
		selfCheck.sendNotification(events)
	}
}

func (selfCheck *SelfCheckWorker) constructUserNotification(events []heartbeatNotificationEvent) ([]*notifier.NotificationPackage, error) {
	type contactEvents struct {
		contact       *moira.ContactData
		events        []moira.NotificationEvent
		triggersTable string
	}

	contactToData := make(map[*moira.ContactData]*contactEvents)

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
				if _, exists := contactToData[contact]; !exists {
					contactToData[contact] = &contactEvents{
						contact: contact,
						events:  []moira.NotificationEvent{},
					}
				}

				contactToData[contact].events = append(contactToData[contact].events, event.NotificationEvent)

				// Build triggers table for this contact if needed
				if event.NotifyAboutEnabledNotifier && contactToData[contact].triggersTable == "" {
					triggersTable := selfCheck.buildTriggersTableForSubscription(subscription)
					if triggersTable != "" {
						contactToData[contact].triggersTable = triggersTable
					}
				}
			}
		}
	}

	notificationPkgs := make([]*notifier.NotificationPackage, 0, len(contactToData))

	for _, data := range contactToData {
		triggerData := moira.TriggerData{
			Name:       "Moira health check",
			ErrorValue: float64(0),
		}

		if data.triggersTable != "" {
			triggerData.Desc = data.triggersTable
		}

		notificationPkgs = append(notificationPkgs, &notifier.NotificationPackage{
			Contact:    *data.contact,
			Trigger:    triggerData,
			Events:     data.events,
			DontResend: true,
		})
	}

	return notificationPkgs, nil
}

func (selfCheck *SelfCheckWorker) buildTriggersTableForSubscription(subscription *moira.SubscriptionData) string {
	if subscription == nil {
		return ""
	}

	triggersTable, err := selfCheck.constructTriggersTable(subscription, selfCheck.Config.Checks.GetUniqueSystemTags())
	if err != nil {
		selfCheck.Logger.Warning().
			Error(err).
			Msg("cannot build triggers table")

		return ""
	}

	if len(triggersTable) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString("These triggers in bad state. Check them:\n")

	for _, link := range triggersTable {
		builder.WriteString("- ")
		builder.WriteString(fmt.Sprintf("[By tags: %s](%s)", strings.Join(link.Tags, "|"), link.Link))
		builder.WriteString("\n")
	}

	return builder.String()
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

type linkResult struct {
	Link  string
	Tags  []string
	Error error
}

type triggersTableElem struct {
	Link string
	Tags []string
}

func (selfCheck *SelfCheckWorker) constructTriggersTable(subscription *moira.SubscriptionData, systemTags []string) ([]triggersTableElem, error) {
	var subscriptionsIds []string

	var err error

	if subscription.TeamID != "" {
		subscriptionsIds, err = selfCheck.Database.GetTeamSubscriptionIDs(subscription.TeamID)
	} else {
		subscriptionsIds, err = selfCheck.Database.GetUserSubscriptionIDs(subscription.User)
	}

	if err != nil {
		return []triggersTableElem{}, err
	}

	table := make([]triggersTableElem, 0)
	tableRows := make(chan linkResult, len(subscriptionsIds))

	var wg sync.WaitGroup
	for _, subId := range subscriptionsIds {
		wg.Add(1)

		go func(subId string) {
			defer func() {
				wg.Done()

				if r := recover(); r != nil {
					selfCheck.Logger.Error().
						Interface("panic", r).
						String("subscriptionId", subId).
						Msg("Panic in goroutine")
					tableRows <- linkResult{"", []string{}, fmt.Errorf("panic: %v", r)}
				}
			}()

			selfCheck.constructLinkToTriggers(subId, systemTags, tableRows)
		}(subId)
	}

	wg.Wait()
	close(tableRows)

	for r := range tableRows {
		if r.Error != nil {
			selfCheck.Logger.Warning().
				Error(r.Error).
				Msg("Failed to construct link to triggers")

			continue
		}

		table = append(table, triggersTableElem{
			Link: r.Link,
			Tags: r.Tags,
		})
	}

	return table, nil
}

func (selfCheck *SelfCheckWorker) constructLinkToTriggers(subscriptionId string, systemTags []string, resultCh chan<- linkResult) {
	sub, err := selfCheck.Database.GetSubscription(subscriptionId)
	if err != nil {
		resultCh <- linkResult{"", []string{}, err}
		return
	}

	if len(moira.Intersect(sub.Tags, systemTags)) > 0 {
		return
	}

	if containsFailedTriggers, e := selfCheck.doesSubscriptionContainsFailedTriigers(&sub); !containsFailedTriggers || e != nil {
		return
	}

	baseUrl, err := url.Parse(selfCheck.Config.FrontURL)
	if err != nil {
		resultCh <- linkResult{"", []string{}, err}
		return
	}

	query := url.Values{}
	query.Add("onlyProblems", "true")

	for i, tag := range sub.Tags {
		query.Add(fmt.Sprintf("tags[%d]", i), tag)
	}

	baseUrl.RawQuery = query.Encode()

	resultCh <- linkResult{baseUrl.String(), sub.Tags, nil}
}

func (selfCheck *SelfCheckWorker) doesSubscriptionContainsFailedTriigers(subscription *moira.SubscriptionData) (bool, error) {
	if subscription == nil || len(subscription.Tags) == 0 {
		return false, nil
	}

	triggerIDs := make(map[string]bool)

	for _, tag := range subscription.Tags {
		ids, err := selfCheck.Database.GetTagTriggerIDs(tag)
		if err != nil {
			return false, fmt.Errorf("failed to get trigger IDs for tag %s: %w", tag, err)
		}

		for _, id := range ids {
			triggerIDs[id] = true
		}
	}

	for triggerID := range triggerIDs {
		checkData, err := selfCheck.Database.GetTriggerLastCheck(triggerID)
		if err != nil {
			continue
		}

		if checkData.State == moira.StateERROR || checkData.State == moira.StateNODATA {
			return true, nil
		}

		for _, metricState := range checkData.Metrics {
			if metricState.State == moira.StateERROR || metricState.State == moira.StateNODATA {
				return true, nil
			}
		}
	}

	return false, nil
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
