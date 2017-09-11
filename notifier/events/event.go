package events

import (
	"time"

	"gopkg.in/tomb.v2"

	"fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics/graphite"
	"github.com/moira-alert/moira/notifier"
)

// FetchEventsWorker checks for new events and new notifications based on it
type FetchEventsWorker struct {
	Logger    moira.Logger
	Database  moira.Database
	Scheduler notifier.Scheduler
	Metrics   *graphite.NotifierMetrics
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
		tags          []string
		triggerData   moira.TriggerData
	)

	if event.State != "TEST" {
		worker.Logger.Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, moira.UseFloat64(event.Value), event.OldState, event.State)

		trigger, err := worker.Database.GetTrigger(event.TriggerID)
		if err != nil {
			return err
		}
		if len(trigger.Tags) == 0 {
			return fmt.Errorf("No tags found for trigger id %s", event.TriggerID)
		}

		triggerData = moira.TriggerData{
			ID:         trigger.ID,
			Name:       trigger.Name,
			Desc:       moira.UseString(trigger.Desc),
			Targets:    trigger.Targets,
			WarnValue:  moira.UseFloat64(trigger.WarnValue),
			ErrorValue: moira.UseFloat64(trigger.ErrorValue),
			Tags:       trigger.Tags,
		}

		tags = append(trigger.Tags, event.GetEventTags()...)
		worker.Logger.Debugf("Getting subscriptions for tags %v", tags)
		subscriptions, err = worker.Database.GetTagsSubscriptions(tags)
		if err != nil {
			return err
		}
	} else {
		worker.Logger.Debugf("Getting subscription id %s for test message", *event.SubscriptionID)
		sub, err := worker.Database.GetSubscription(*event.SubscriptionID)
		if err != nil {
			worker.Metrics.SubsMalformed.Mark(1)
			return err
		}
		subscriptions = []*moira.SubscriptionData{&sub}
	}

	duplications := make(map[string]bool)
	for _, subscription := range subscriptions {
		if subscription != nil && (event.State == "TEST" || (subscription.Enabled && subset(subscription.Tags, tags))) {
			worker.Logger.Debugf("Processing contact ids %v for subscription %s", subscription.Contacts, subscription.ID)
			for _, contactID := range subscription.Contacts {
				contact, err := worker.Database.GetContact(contactID)
				if err != nil {
					worker.Logger.Warning(err.Error())
					continue
				}
				event.SubscriptionID = &subscription.ID
				notification := worker.Scheduler.ScheduleNotification(time.Now(), event, triggerData, contact, false, 0)
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

		} else if subscription == nil {
			worker.Logger.Debugf("Subscription is nil")
		} else if !subscription.Enabled {
			worker.Logger.Debugf("Subscription %s is disabled", subscription.ID)
		} else {
			worker.Logger.Debugf("Subscription %s has extra tags", subscription.ID)
		}
	}
	return nil
}

func subset(first, second []string) bool {
	set := make(map[string]bool)
	for _, value := range second {
		set[value] = true
	}

	for _, value := range first {
		if !set[value] {
			return false
		}
	}

	return true
}
