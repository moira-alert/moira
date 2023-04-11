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
					worker.Logger.Info().Msg("Moira Notifier Fetching events stopped")
					return nil
				}
			default:
				{
					event, err := worker.Database.FetchNotificationEvent()
					if err != nil {
						if err != database.ErrNil {
							worker.Metrics.EventsMalformed.Mark(1)
							worker.Logger.Warning().
								Error(err).
								Msg("Failed to fetch notification event")
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
						worker.Logger.Error().
							Error(err).
							String("trigger_id", event.TriggerID).
							String("contact_id", event.ContactID).
							Msg("Failed processEvent")
					}
				}
			}
		}
	})
	worker.Logger.Info().Msg("Moira Notifier Fetching events started")
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
		log.Debug().
			String("metric", fmt.Sprintf("%s == %s", event.Metric, event.GetMetricsValues())).
			String("old_state", event.OldState.String()).
			String("new_state", event.State.String()).
			Msg("Processing trigger for metric")

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

		log.Debug().
			Interface("trigger_tags", trigger.Tags).
			Msg("Getting subscriptions for given tags")

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
					contactLogger.Warning().
						Error(err).
						Msg("Failed to get contact, skip handling it")
					continue
				}
				event.SubscriptionID = &subscription.ID
				notification := worker.Scheduler.ScheduleNotification(time.Now(), event, triggerData,
					contact, subscription.Plotting, false, 0, contactLogger)
				key := notification.GetKey()
				if _, exist := duplications[key]; !exist {
					if err := worker.Database.AddNotification(notification); err != nil {
						contactLogger.Error().
							Error(err).
							Msg("Failed to save scheduled notification")
					}
					duplications[key] = true
				} else {
					contactLogger.Debug().
						Interface("contact", notification.Contact).
						Msg("Skip duplicated notification for a contact")
				}
			}
		}
	}
	return nil
}

func (worker *FetchEventsWorker) getNotificationSubscriptions(event moira.NotificationEvent, logger moira.Logger) (*moira.SubscriptionData, error) {
	if event.SubscriptionID != nil {
		subID := moira.UseString(event.SubscriptionID)
		logger.Debug().
			String(moira.LogFieldNameSubscriptionID, subID).
			Msg("Getting subscription for test message")

		notifier.SetLogLevelByConfig(worker.Config.LogSubscriptionsToLevel, subID, &logger)
		sub, err := worker.Database.GetSubscription(*event.SubscriptionID)
		if err != nil {
			worker.Metrics.SubsMalformed.Mark(1)
			return nil, fmt.Errorf("error while read subscription %s: %s", *event.SubscriptionID, err.Error())
		}
		return &sub, nil
	} else if event.ContactID != "" {
		logger.Debug().
			String(moira.LogFieldNameContactID, event.ContactID).
			Msg("Getting contact for test message")

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
		logger.Debug().Msg("Subscription is nil")
		return false
	}
	if event.State != moira.StateTEST {
		if !subscription.Enabled {
			logger.Debug().Msg("Subscription is disabled")
			return false
		}
		if subscription.MustIgnore(&event) {
			logger.Debug().
				String("ignored_transaction", fmt.Sprintf("%s -> %s", event.OldState, event.State)).
				Msg("Subscription is managed to ignore specific transitions")
			return false
		}
		if !moira.Subset(subscription.Tags, trigger.Tags) {
			return false
		}
	}
	return true
}
