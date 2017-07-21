package events

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/moira-alert/moira-alert/notifier"
	"sync"
	"time"
)

//FetchEventsWorker - check for new events and new notifications based on it
type FetchEventsWorker struct {
	logger    moira.Logger
	database  moira.Database
	scheduler notifier.Scheduler
	metrics   *graphite.NotifierMetrics
}

//NewFetchEventWorker new worker
func NewFetchEventWorker(database moira.Database, logger moira.Logger, metrics *graphite.NotifierMetrics) *FetchEventsWorker {
	return &FetchEventsWorker{
		logger:    logger,
		database:  database,
		scheduler: notifier.NewScheduler(database, logger),
		metrics:   metrics,
	}
}

// Run is a cycle that fetches events from database
func (worker *FetchEventsWorker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	worker.logger.Debug("Start Fetching Events")
	for {
		select {
		case <-shutdown:
			{
				worker.logger.Debug("Stop Fetching Events")
				return
			}
		default:
			{
				event, err := worker.database.FetchEvent()
				if err != nil {
					worker.metrics.EventsMalformed.Mark(1)
					continue
				}
				if event != nil {
					worker.metrics.EventsReceived.Mark(1)
					if err := worker.processEvent(*event); err != nil {
						worker.metrics.EventsProcessingFailed.Mark(1)
						worker.logger.Errorf("Failed processEvent. %s", err)
					}
				}
			}
		}
	}
}

func (worker *FetchEventsWorker) processEvent(event moira.EventData) error {
	var (
		subscriptions []moira.SubscriptionData
		tags          []string
		trigger       moira.TriggerData
		err           error
	)

	if event.State != "TEST" {
		worker.logger.Debugf("Processing trigger id %s for metric %s == %f, %s -> %s", event.TriggerID, event.Metric, event.Value, event.OldState, event.State)

		trigger, err = worker.database.GetNotificationTrigger(event.TriggerID)
		if err != nil {
			return err
		}

		tags, err = worker.database.GetTriggerTags(event.TriggerID)
		if err != nil {
			return err
		}
		trigger.Tags = tags
		tags = append(tags, event.GetEventTags()...)

		worker.logger.Debugf("Getting subscriptions for tags %v", tags)
		subscriptions, err = worker.database.GetTagsSubscriptions(tags)
		if err != nil {
			return err
		}
	} else {
		worker.logger.Debugf("Getting subscription id %s for test message", event.SubscriptionID)
		sub, err := worker.database.GetSubscription(event.SubscriptionID)
		if err != nil {
			return err
		}
		subscriptions = []moira.SubscriptionData{sub}
	}

	duplications := make(map[string]bool)
	for _, subscription := range subscriptions {
		if event.State == "TEST" || (subscription.Enabled && subset(subscription.Tags, tags)) {
			worker.logger.Debugf("Processing contact ids %v for subscription %s", subscription.Contacts, subscription.ID)
			for _, contactID := range subscription.Contacts {
				contact, err := worker.database.GetContact(contactID)
				if err != nil {
					worker.logger.Warning(err.Error())
					continue
				}
				event.SubscriptionID = subscription.ID
				notification := worker.scheduler.ScheduleNotification(time.Now(), event, trigger, contact, false, 0)
				key := notification.GetKey()
				if _, exist := duplications[key]; !exist {
					if err := worker.database.AddNotification(notification); err != nil {
						worker.logger.Errorf("Failed to save scheduled notification: %s", err)
					}
					duplications[key] = true
				} else {
					worker.logger.Debugf("Skip duplicated notification for contact %s", notification.Contact)
				}
			}
		} else if !subscription.Enabled {
			worker.logger.Debugf("Subscription %s is disabled", subscription.ID)
		} else {
			worker.logger.Debugf("Subscription %s has extra tags", subscription.ID)
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
