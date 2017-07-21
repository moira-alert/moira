package moira

import (
	"sync"
	"time"
)

// Database implements DB functionality
type Database interface {
	FetchEvent() (*EventData, error)
	GetNotificationTrigger(id string) (TriggerData, error)
	GetTriggerTags(id string) ([]string, error)
	GetTagsSubscriptions(tags []string) ([]SubscriptionData, error)
	GetSubscription(id string) (SubscriptionData, error)
	GetContact(id string) (ContactData, error)
	GetContacts() ([]ContactData, error)
	SetContact(contact *ContactData) error
	AddNotification(notification *ScheduledNotification) error
	GetTriggerThrottlingTimestamps(id string) (time.Time, time.Time)
	GetTriggerEventsCount(id string, from int64) int64
	SetTriggerThrottlingTimestamp(id string, next time.Time) error
	GetNotifications(to int64) ([]*ScheduledNotification, error)
	GetMetricsCount() (int64, error)
	GetChecksCount() (int64, error)

	UpdateMetricsHeartbeat() error
	GetPatterns() ([]string, error)
	SaveMetrics(buffer map[string]*MatchedMetric) error

	GetUserSubscriptions(string) ([]string, error)
	GetUserContacts(string) ([]string, error)

	GetTagNames() ([]string, error)
	GetTags([]string) (map[string]TagData, error)
	GetTag(string) (TagData, error)

	GetTriggerIds() ([]string, int64, error)
	GetFilteredTriggersIds([]string, bool) ([]string, int64, error)
	GetTrigger(string) (*Trigger, error)
	GetTriggersChecks([]string) ([]TriggerChecks, error)
	GetTriggerLastCheck(string) (*CheckData, error)

	GetEvents(string, int64, int64) ([]*EventData, error)
}

// Logger implements logger abstraction
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// Worker interface for implementing specified parallel workers
type Worker interface {
	Run(shutdown chan bool, wg *sync.WaitGroup)
}

// Sender interface for implementing specified contact type sender
type Sender interface {
	SendEvents(events EventsData, contact ContactData, trigger TriggerData, throttled bool) error
	Init(senderSettings map[string]string, logger Logger) error
}
