package moira

import (
	"time"

	"github.com/beevee/go-chart"
	"gopkg.in/tomb.v2"
)

// Database implements DB functionality
type Database interface {
	// SelfState
	UpdateMetricsHeartbeat() error
	GetMetricsUpdatesCount() (int64, error)
	GetChecksUpdatesCount() (int64, error)
	GetRemoteChecksUpdatesCount() (int64, error)
	GetNotifierState() (string, error)
	SetNotifierState(string) error

	// Tag storing
	GetTagNames() ([]string, error)
	RemoveTag(tagName string) error
	GetTagTriggerIDs(tagName string) ([]string, error)

	// LastCheck storing
	GetTriggerLastCheck(triggerID string) (CheckData, error)
	SetTriggerLastCheck(triggerID string, checkData *CheckData, isRemote bool) error
	RemoveTriggerLastCheck(triggerID string) error
	SetTriggerCheckMaintenance(triggerID string, metrics map[string]int64, triggerMaintenance *int64, userLogin string, timeCallMaintenance int64) error

	// Trigger storing
	GetLocalTriggerIDs() ([]string, error)
	GetAllTriggerIDs() ([]string, error)
	GetRemoteTriggerIDs() ([]string, error)
	GetTrigger(triggerID string) (Trigger, error)
	GetTriggers(triggerIDs []string) ([]*Trigger, error)
	GetTriggerChecks(triggerIDs []string) ([]*TriggerCheck, error)
	SaveTrigger(triggerID string, trigger *Trigger) error
	RemoveTrigger(triggerID string) error
	GetPatternTriggerIDs(pattern string) ([]string, error)
	RemovePatternTriggerIDs(pattern string) error

	// Throttling
	GetTriggerThrottling(triggerID string) (time.Time, time.Time)
	SetTriggerThrottling(triggerID string, next time.Time) error
	DeleteTriggerThrottling(triggerID string) error

	// NotificationEvent storing
	GetNotificationEvents(triggerID string, start, size int64) ([]*NotificationEvent, error)
	PushNotificationEvent(event *NotificationEvent, ui bool) error
	GetNotificationEventCount(triggerID string, from int64) int64
	FetchNotificationEvent() (NotificationEvent, error)
	RemoveAllNotificationEvents() error

	// ContactData storing
	GetContact(contactID string) (ContactData, error)
	GetContacts(contactIDs []string) ([]*ContactData, error)
	GetAllContacts() ([]*ContactData, error)
	RemoveContact(contactID string) error
	SaveContact(contact *ContactData) error
	GetUserContactIDs(userLogin string) ([]string, error)

	// SubscriptionData storing
	GetSubscription(id string) (SubscriptionData, error)
	GetSubscriptions(subscriptionIDs []string) ([]*SubscriptionData, error)
	SaveSubscription(subscription *SubscriptionData) error
	SaveSubscriptions(subscriptions []*SubscriptionData) error
	RemoveSubscription(subscriptionID string) error
	GetUserSubscriptionIDs(userLogin string) ([]string, error)
	GetTagsSubscriptions(tags []string) ([]*SubscriptionData, error)

	// ScheduledNotification storing
	GetNotifications(start, end int64) ([]*ScheduledNotification, int64, error)
	RemoveNotification(notificationKey string) (int64, error)
	RemoveAllNotifications() error
	FetchNotifications(to int64) ([]*ScheduledNotification, error)
	AddNotification(notification *ScheduledNotification) error
	AddNotifications(notification []*ScheduledNotification, timestamp int64) error

	// Patterns and metrics storing
	GetPatterns() ([]string, error)
	AddPatternMetric(pattern, metric string) error
	GetPatternMetrics(pattern string) ([]string, error)
	RemovePattern(pattern string) error
	RemovePatternsMetrics(pattern []string) error
	RemovePatternWithMetrics(pattern string) error

	SubscribeMetricEvents(tomb *tomb.Tomb) (<-chan *MetricEvent, error)
	SaveMetrics(buffer map[string]*MatchedMetric) error
	GetMetricRetention(metric string) (int64, error)
	GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*MetricValue, error)
	RemoveMetricValues(metric string, toTime int64) error
	RemoveMetricsValues(metrics []string, toTime int64) error

	AddLocalTriggersToCheck(triggerIDs []string) error
	GetLocalTriggersToCheck(count int) ([]string, error)
	GetLocalTriggersToCheckCount() (int64, error)

	AddRemoteTriggersToCheck(triggerIDs []string) error
	GetRemoteTriggersToCheck(count int) ([]string, error)
	GetRemoteTriggersToCheckCount() (int64, error)

	// TriggerCheckLock storing
	AcquireTriggerCheckLock(triggerID string, timeout int) error
	DeleteTriggerCheckLock(triggerID string) error
	SetTriggerCheckLock(triggerID string) (bool, error)

	// Bot data storing
	GetIDByUsername(messenger, username string) (string, error)
	SetUsernameID(messenger, username, id string) error
	RemoveUser(messenger, username string) error

	// Triggers without subscription manipulation
	MarkTriggersAsUnused(triggerIDs ...string) error
	GetUnusedTriggerIDs() ([]string, error)
	MarkTriggersAsUsed(triggerIDs ...string) error

	// Triggers to reindex in full-text search index
	FetchTriggersToReindex(from int64) ([]string, error)
	RemoveTriggersToReindex(to int64) error

	// Creates Lock
	NewLock(name string, ttl time.Duration) Lock
}

// Lock implements lock abstraction
type Lock interface {
	Acquire(stop <-chan struct{}) (lost <-chan struct{}, error error)
	Release()
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
	SendEvents(events NotificationEvents, contact ContactData, trigger TriggerData, plot []byte, throttled bool) error
	Init(senderSettings map[string]string, logger Logger, location *time.Location, dateTimeFormat string) error
}

// ImageStore is the interface for image storage providers
type ImageStore interface {
	StoreImage(image []byte) (string, error)
	IsEnabled() bool
}

// Searcher interface implements full-text search index functionality
type Searcher interface {
	Start() error
	Stop() error
	IsReady() bool
	SearchTriggers(filterTags []string, searchString string, onlyErrors bool,
		page int64, size int64) (searchResults []*SearchResult, total int64, err error)
}

// PlotTheme is an interface to access plot theme styles
type PlotTheme interface {
	GetTitleStyle() chart.Style
	GetGridStyle() chart.Style
	GetCanvasStyle() chart.Style
	GetBackgroundStyle(maxMarkLen int) chart.Style
	GetThresholdStyle(thresholdType string) chart.Style
	GetAnnotationStyle(thresholdType string) chart.Style
	GetSerieStyles(curveInd int) (curveStyle, pointStyle chart.Style)
	GetLegendStyle() chart.Style
	GetXAxisStyle() chart.Style
	GetYAxisStyle() chart.Style
}
