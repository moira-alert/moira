package events

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira/metrics"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/notifier"
)

// FetchEventsWorker checks for new events and new notifications based on it
type FetchEventsWorker struct {
	Logger    moira.Logger
	Database  moira.Database
	Scheduler notifier.Scheduler
	Metrics   *metrics.NotifierMetrics
	Config    notifier.Config
	tomb      tomb.Tomb
}

// Start is a cycle that fetches events from database
func (worker *FetchEventsWorker) Start() {
	worker.tomb.Go(func() error {
		for {
			select {
			case <-worker.tomb.Dying():
				{
					worker.Logger.Info("Moira Notifier Fetching events stopped")
					return nil
				}
			default:
				{
					event, err := worker.Database.FetchNotificationEvent()
					if err != nil {
						if err != database.ErrNil {
							worker.Metrics.EventsMalformed.Mark(1)
							worker.Logger.WarningWithError("Failed to fetch notification event", err)
							time.Sleep(time.Second * 5) //nolint
						}
						continue
					}

					worker.Metrics.EventsReceived.Mark(1)
					stateMeterName := event.OldState.String() + "_to_" + event.State.String()
					stateMeter, found := worker.Metrics.EventsByState.GetRegisteredMeter(stateMeterName)
					if !found {
						stateMeter = worker.Metrics.EventsByState.RegisterMeter(stateMeterName, "events", "bystate", stateMeterName)
					}
					stateMeter.Mark(1)

					if err := worker.processEvent(event); err != nil {
						worker.Metrics.EventsProcessingFailed.Mark(1)
						worker.Logger.ErrorWithError("Failed processEvent", err)
					}
				}
			}
		}
	})
	worker.Logger.Info("Moira Notifier Fetching events started")
}

// Stop stops new event fetching and wait for finish
func (worker *FetchEventsWorker) Stop() error {
	worker.tomb.Kill(nil)
	return worker.tomb.Wait()
}

func (worker *FetchEventsWorker) processEvent(event moira.NotificationEvent) error {
	log := worker.Logger.Clone().
		String(moira.LogFieldNameTriggerID, event.TriggerID)

	var (
		subscriptions []*moira.SubscriptionData
		triggerData   moira.TriggerData
	)
	if event.State != moira.StateTEST {
		log.Debugf("Processing trigger for metric %s == %f, %s -> %s",
			event.Metric, event.GetMetricsValues(), event.OldState, event.State)

		trigger, err := worker.Database.GetTrigger(event.TriggerID)
		if err != nil {
			return err
		}
		if len(trigger.Tags) == 0 {
			return fmt.Errorf("no tags found for trigger id %s", event.TriggerID)
		}

		triggerData = moira.TriggerData{
			ID:         trigger.ID,
			Name:       trigger.Name,
			Desc:       moira.UseString(trigger.Desc),
			Targets:    trigger.Targets,
			WarnValue:  moira.UseFloat64(trigger.WarnValue),
			ErrorValue: moira.UseFloat64(trigger.ErrorValue),
			IsRemote:   trigger.IsRemote,
			Tags:       trigger.Tags,
		}

		log.Debugf("Getting subscriptions for tags %v", trigger.Tags)
		subscriptions, err = worker.Database.GetTagsSubscriptions(trigger.Tags)
		if err != nil {
			return err
		}
	} else {
		sub, err := worker.getNotificationSubscriptions(event, log)
		if err != nil {
			return err
		}
		subscriptions = []*moira.SubscriptionData{sub}
	}

	duplications := make(map[string]bool)

	for _, subscription := range subscriptions {
		subLogger := log.Clone()
		if subscription != nil {
			subLogger.String(moira.LogFieldNameSubscriptionID, subscription.ID)
			notifier.SetLogLevelByConfig(worker.Config.LogSubscriptionsToLevel, subscription.ID, &subLogger)
		}
		if worker.isNotificationRequired(subscription, triggerData, event, subLogger) {
			for _, contactID := range subscription.Contacts {
				contactLogger := subLogger.Clone().
					String(moira.LogFieldNameContactID, contactID)
				notifier.SetLogLevelByConfig(worker.Config.LogContactsToLevel, contactID, &contactLogger)
				contact, err := worker.Database.GetContact(contactID)
				if err != nil {
					contactLogger.WarningWithError("Failed to get contact, skip handling it, error", err)
					continue
				}
				event.SubscriptionID = &subscription.ID
				notification := worker.Scheduler.ScheduleNotification(time.Now(), event, triggerData,
					contact, subscription.Plotting, false, 0, contactLogger)
				key := notification.GetKey()
				if _, exist := duplications[key]; !exist {
					if err := worker.Database.AddNotification(notification); err != nil {
						contactLogger.ErrorWithError("Failed to save scheduled notification", err)
					}
					duplications[key] = true
				} else {
					contactLogger.Debugf("Skip duplicated notification for contact %s", notification.Contact)
				}
			}
		}
	}
	return nil
}

func (worker *FetchEventsWorker) getNotificationSubscriptions(event moira.NotificationEvent, logger moira.Logger) (*moira.SubscriptionData, error) {
	if event.SubscriptionID != nil {
		subID := moira.UseString(event.SubscriptionID)
		logger.Clone().
			String(moira.LogFieldNameSubscriptionID, subID).
			Debug("Getting subscription for test message")
		notifier.SetLogLevelByConfig(worker.Config.LogSubscriptionsToLevel, subID, &logger)
		sub, err := worker.Database.GetSubscription(*event.SubscriptionID)
		if err != nil {
			worker.Metrics.SubsMalformed.Mark(1)
			return nil, fmt.Errorf("error while read subscription %s: %s", *event.SubscriptionID, err.Error())
		}
		return &sub, nil
	} else if event.ContactID != "" {
		logger.Clone().
			String(moira.LogFieldNameContactID, event.ContactID).
			Debug("Getting contact for test message")
		notifier.SetLogLevelByConfig(worker.Config.LogContactsToLevel, event.ContactID, &logger)

		contact, err := worker.Database.GetContact(event.ContactID)
		if err != nil {
			return nil, fmt.Errorf("error while read contact %s: %s", event.ContactID, err.Error())
		}
		sub := &moira.SubscriptionData{
			ID:                "testSubscription",
			User:              contact.User,
			TeamID:            contact.Team,
			ThrottlingEnabled: false,
			Enabled:           true,
			Tags:              make([]string, 0),
			Contacts:          []string{contact.ID},
			Schedule:          moira.ScheduleData{},
		}
		return sub, nil
	}

	return nil, nil
}

func (worker *FetchEventsWorker) isNotificationRequired(subscription *moira.SubscriptionData, trigger moira.TriggerData,
	event moira.NotificationEvent, logger moira.Logger) bool {
	if subscription == nil {
		logger.Debug("Subscription is nil")
		return false
	}
	if event.State != moira.StateTEST {
		if !subscription.Enabled {
			logger.Debug("Subscription is disabled")
			return false
		}
		if subscription.MustIgnore(&event) {
			logger.Debugf("Subscription is managed to ignore %s -> %s transitions", event.OldState, event.State)
			return false
		}
		if !moira.Subset(subscription.Tags, trigger.Tags) {
			return false
		}
	}
	return true
}
