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
							worker.Logger.Warning(err)
							time.Sleep(time.Second * 5)
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
						worker.Logger.Errorf("Failed processEvent. %s", err)
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
	var (
		subscriptions []*moira.SubscriptionData
		triggerData   moira.TriggerData
	)

	if event.State != moira.StateTEST {
		worker.Logger.Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, event.GetMetricsValues(), event.OldState, event.State)

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

		worker.Logger.Debugf("Getting subscriptions for tags %v", trigger.Tags)
		subscriptions, err = worker.Database.GetTagsSubscriptions(trigger.Tags)
		if err != nil {
			return err
		}
	} else {
		sub, err := worker.getNotificationSubscriptions(event)
		if err != nil {
			return err
		}
		subscriptions = []*moira.SubscriptionData{sub}
	}

	duplications := make(map[string]bool)

	for _, subscription := range subscriptions {
		if worker.isNotificationRequired(subscription, triggerData, event) {
			for _, contactID := range subscription.Contacts {
				contact, err := worker.Database.GetContact(contactID)
				if err != nil {
					worker.Logger.Warningf("Failed to get contact: %s, skip handling it, error: %v", contactID, err)
					continue
				}
				event.SubscriptionID = &subscription.ID
				notification := worker.Scheduler.ScheduleNotification(time.Now(), event, triggerData,
					contact, subscription.Plotting, false, 0)
				key := notification.GetKey()
				if _, exist := duplications[key]; !exist {
					if err := worker.Database.AddNotification(notification); err != nil {
						worker.Logger.Errorf("Failed to save scheduled notification: %s", err)
					}
					duplications[key] = true
				} else {
					worker.Logger.Debugf("Skip duplicated notification for contact %s", notification.Contact)
				}
			}
		}
	}
	return nil
}

func (worker *FetchEventsWorker) getNotificationSubscriptions(event moira.NotificationEvent) (*moira.SubscriptionData, error) {
	if event.SubscriptionID != nil {
		worker.Logger.Debugf("Getting subscriptionID %s for test message", *event.SubscriptionID)
		sub, err := worker.Database.GetSubscription(*event.SubscriptionID)
		if err != nil {
			worker.Metrics.SubsMalformed.Mark(1)
			return nil, fmt.Errorf("error while read subscription %s: %s", *event.SubscriptionID, err.Error())
		}
		return &sub, nil
	} else if event.ContactID != "" {
		worker.Logger.Debugf("Getting contactID %s for test message", event.ContactID)
		contact, err := worker.Database.GetContact(event.ContactID)
		if err != nil {
			return nil, fmt.Errorf("error while read contact %s: %s", event.ContactID, err.Error())
		}
		sub := &moira.SubscriptionData{
			ID:                "testSubscription",
			User:              contact.User,
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

func (worker *FetchEventsWorker) isNotificationRequired(subscription *moira.SubscriptionData, trigger moira.TriggerData, event moira.NotificationEvent) bool {
	if subscription == nil {
		worker.Logger.Debugf("Subscription is nil")
		return false
	}
	if event.State != moira.StateTEST {
		if !subscription.Enabled {
			worker.Logger.Debugf("Subscription %s is disabled", subscription.ID)
			return false
		}
		if subscription.MustIgnore(&event) {
			worker.Logger.Debugf("Subscription %s is managed to ignore %s -> %s transitions", subscription.ID, event.OldState, event.State)
			return false
		}
		if !moira.Subset(subscription.Tags, trigger.Tags) {
			return false
		}
	}
	return true
}
