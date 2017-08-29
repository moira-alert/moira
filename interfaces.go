package moira

import (
	"gopkg.in/tomb.v2"
	"time"
)

// Database implements DB functionality
type Database interface {
	//SelfState
	UpdateMetricsHeartbeat() error
	GetMetricsCount() (int64, error)
	GetChecksCount() (int64, error)

	//Tag storing
	GetTagNames() ([]string, error)
	RemoveTag(tagName string) error
	GetTagTriggerIDs(tagName string) ([]string, error)

	//LastCheck storing
	GetTriggerLastCheck(triggerID string) (*CheckData, error)
	SetTriggerLastCheck(triggerID string, checkData *CheckData) error
	GetTriggerCheckIDs() ([]string, int64, error)

	GetTriggerIds() ([]string, error)
	GetFilteredTriggerCheckIds([]string, bool) ([]string, int64, error)
	GetTrigger(string) (*Trigger, error)
	GetNotificationTrigger(id string) (TriggerData, error)
	GetTriggerChecks(triggerCheckIds []string) ([]TriggerChecks, error)
	SetTriggerMetricsMaintenance(triggerId string, metrics map[string]int64) error
	GetPatternTriggerIds(pattern string) ([]string, error)
	GetTriggers(triggerIds []string) ([]*Trigger, error)
	DeleteTriggerThrottling(triggerId string) error
	DeleteTrigger(triggerId string) error
	SaveTrigger(triggerId string, trigger *Trigger) error
	RemovePatternTriggers(pattern string) error
	GetTriggerTags(id string) ([]string, error)
	GetTriggerThrottlingTimestamps(id string) (time.Time, time.Time)
	SetTriggerThrottlingTimestamp(id string, next time.Time) error

	AddTriggerToCheck(triggerId string) error
	GetTriggerToCheck() (*string, error)

	//NotificationEvent storing
	GetNotificationEvents(triggerID string, start, size int64) ([]*NotificationEvent, error)
	PushNotificationEvent(event *NotificationEvent, ui bool) error
	GetNotificationEventCount(triggerID string, from int64) int64
	FetchNotificationEvent() (NotificationEvent, error)

	//ContactData storing
	GetContact(contactID string) (ContactData, error)
	GetContacts(contactIDs []string) ([]*ContactData, error)
	GetAllContacts() ([]*ContactData, error)
	WriteContact(contact *ContactData) error
	RemoveContact(string, string) error
	SaveContact(contact *ContactData) error
	GetUserContactIDs(userLogin string) ([]string, error)

	//SubscriptionData storing
	GetSubscription(id string) (SubscriptionData, error)
	GetSubscriptions(subscriptionIds []string) ([]*SubscriptionData, error)
	WriteSubscriptions(subscriptions []*SubscriptionData) error
	SaveSubscription(subscription *SubscriptionData) error
	RemoveSubscription(subscriptionId string, userLogin string) error
	GetUserSubscriptionIDs(userLogin string) ([]string, error)
	GetTagsSubscriptions(tags []string) ([]*SubscriptionData, error)

	//ScheduledNotification storing
	GetNotifications(start, end int64) ([]*ScheduledNotification, int64, error)
	RemoveNotification(notificationKey string) (int64, error)
	GetNotificationsAndDelete(to int64) ([]*ScheduledNotification, error)
	AddNotification(notification *ScheduledNotification) error
	AddNotifications(notification []*ScheduledNotification, timestamp int64) error

	//Patterns and metrics storing
	GetPatterns() ([]string, error)
	GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*MetricValue, error)
	SaveMetrics(buffer map[string]*MatchedMetric) error
	SubscribeMetricEvents(tomb *tomb.Tomb) <-chan *MetricEvent
	GetMetricRetention(metric string) (int64, error)
	AddPatternMetric(pattern, metric string) error
	GetPatternMetrics(pattern string) ([]string, error)
	RemovePattern(pattern string) error
	RemovePatternsMetrics(pattern []string) error
	RemovePatternWithMetrics(pattern string) error
	RemoveMetricValues(metric string, toTime int64) error

	//TriggerCheckLock storing
	AcquireTriggerCheckLock(triggerID string, timeout int) error
	DeleteTriggerCheckLock(triggerID string) error
	SetTriggerCheckLock(triggerID string) (bool, error)
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

// Sender interface for implementing specified contact type sender
type Sender interface {
	SendEvents(events EventsData, contact ContactData, trigger TriggerData, throttled bool) error
	Init(senderSettings map[string]string, logger Logger) error
}
