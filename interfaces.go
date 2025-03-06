package moira

import (
	"time"

	"github.com/moira-alert/go-chart"
	"github.com/moira-alert/moira/logging"
	"gopkg.in/tomb.v2"
)

// Database implements DB functionality.
type Database interface {
	// SelfState
	UpdateMetricsHeartbeat() error
	GetMetricsUpdatesCount() (int64, error)
	GetChecksUpdatesCount() (int64, error)
	GetRemoteChecksUpdatesCount() (int64, error)
	GetPrometheusChecksUpdatesCount() (int64, error)
	GetNotifierState() (string, error)
	SetNotifierState(string) error

	// Tag storing
	GetTagNames() ([]string, error)
	GetSystemTagNames() ([]string, error)
	SyncSystemTags(tags []string) error
	CreateTags(tags []string) error
	RemoveTag(tagName string) error
	GetTagTriggerIDs(tagName string) ([]string, error)
	CleanUpAbandonedTags() (int, error)

	// LastCheck storing
	GetTriggerLastCheck(triggerID string) (CheckData, error)
	SetTriggerLastCheck(triggerID string, checkData *CheckData, clusterKey ClusterKey) error
	RemoveTriggerLastCheck(triggerID string) error
	SetTriggerCheckMaintenance(triggerID string, metrics map[string]int64, triggerMaintenance *int64, userLogin string, timeCallMaintenance int64) error
	CleanUpAbandonedTriggerLastCheck() error

	// Trigger storing
	GetAllTriggerIDs() ([]string, error)
	GetTriggerIDs(clusterKey ClusterKey) ([]string, error)

	GetTriggerCount(clusterKeys []ClusterKey) (map[ClusterKey]int64, error)

	GetTrigger(triggerID string) (Trigger, error)
	GetTriggers(triggerIDs []string) ([]*Trigger, error)
	GetTriggerChecks(triggerIDs []string) ([]*TriggerCheck, error)
	SaveTrigger(triggerID string, trigger *Trigger) error
	RemoveTrigger(triggerID string) error
	GetPatternTriggerIDs(pattern string) ([]string, error)
	RemovePatternTriggerIDs(pattern string) error
	GetTriggerIDsStartWith(prefix string) ([]string, error)

	// SearchResult AKA pager storing
	GetTriggersSearchResults(searchResultsID string, page, size int64) ([]*SearchResult, int64, error)
	SaveTriggersSearchResults(searchResultsID string, searchResults []*SearchResult, recordTTL time.Duration) error
	IsTriggersSearchResultsExist(pagerID string) (bool, error)
	DeleteTriggersSearchResults(pagerID string) error

	// Throttling
	GetTriggerThrottling(triggerID string) (time.Time, time.Time)
	SetTriggerThrottling(triggerID string, next time.Time) error
	DeleteTriggerThrottling(triggerID string) error

	// NotificationEvent storing
	GetNotificationEvents(triggerID string, page, size int64, from, to string) ([]*NotificationEvent, error)
	PushNotificationEvent(event *NotificationEvent, ui bool) error
	GetNotificationEventCount(triggerID string, from, to string) int64
	FetchNotificationEvent() (NotificationEvent, error)
	RemoveAllNotificationEvents() error

	// ContactData storing
	GetContact(contactID string) (ContactData, error)
	GetContacts(contactIDs []string) ([]*ContactData, error)
	GetAllContacts() ([]*ContactData, error)
	RemoveContact(contactID string) error
	SaveContact(contact *ContactData) error
	GetUserContactIDs(userLogin string) ([]string, error)
	GetTeamContactIDs(teamID string) ([]string, error)

	// SubscriptionData storing
	GetSubscription(id string) (SubscriptionData, error)
	GetSubscriptions(subscriptionIDs []string) ([]*SubscriptionData, error)
	SaveSubscription(subscription *SubscriptionData) error
	SaveSubscriptions(subscriptions []*SubscriptionData) error
	RemoveSubscription(subscriptionID string) error
	GetUserSubscriptionIDs(userLogin string) ([]string, error)
	GetTeamSubscriptionIDs(teamID string) ([]string, error)
	GetTagsSubscriptions(tags []string) ([]*SubscriptionData, error)

	// ScheduledNotification storing
	GetNotifications(start, end int64) ([]*ScheduledNotification, int64, error)
	GetNotificationsHistoryByContactID(contactID string, from, to, page, size int64) ([]*NotificationEventHistoryItem, error)
	RemoveNotification(notificationKey string) (int64, error)
	RemoveAllNotifications() error
	FetchNotifications(to int64, limit int64) ([]*ScheduledNotification, error)
	AddNotification(notification *ScheduledNotification) error
	AddNotifications(notification []*ScheduledNotification, timestamp int64) error
	PushContactNotificationToHistory(notification *ScheduledNotification) error
	CleanUpOutdatedNotificationHistory(ttl int64) error
	CountEventsInNotificationHistory(contactIDs []string, from, to string) ([]*ContactIDWithNotificationCount, error)

	// Patterns and metrics storing
	GetPatterns() ([]string, error)
	AddPatternMetric(pattern, metric string) error
	GetPatternMetrics(pattern string) ([]string, error)
	RemovePattern(pattern string) error
	RemovePatternsMetrics(pattern []string) error
	RemovePatternWithMetrics(pattern string) error

	SubscribeMetricEvents(tomb *tomb.Tomb, params *SubscribeMetricEventsParams) (<-chan *MetricEvent, error)
	SaveMetrics(buffer map[string]*MatchedMetric) error
	GetMetricRetention(metric string) (int64, error)
	GetMetricsValues(metrics []string, from int64, until int64) (map[string][]*MetricValue, error)
	RemoveMetricRetention(metric string) error
	RemoveMetricValues(metric string, from, to string) (int64, error)
	RemoveMetricsValues(metrics []string, toTime int64) error
	GetMetricsTTLSeconds() int64

	AddTriggersToCheck(clusterKey ClusterKey, triggerIDs []string) error
	GetTriggersToCheck(clusterKey ClusterKey, count int) ([]string, error)
	GetTriggersToCheckCount(clusterKey ClusterKey) (int64, error)

	// TriggerCheckLock storing
	AcquireTriggerCheckLock(triggerID string, maxAttemptsCount int) error
	DeleteTriggerCheckLock(triggerID string) error
	SetTriggerCheckLock(triggerID string) (bool, error)
	ReleaseTriggerCheckLock(triggerID string)

	// Bot data storing
	GetChatByUsername(messenger, username string) (string, error)
	SetUsernameChat(messenger, username, chatRaw string) error
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

	// Teams management
	SaveTeam(teamID string, team Team) error
	GetTeam(teamID string) (Team, error)
	GetTeamByName(name string) (Team, error)
	GetAllTeams() ([]Team, error)
	SaveTeamsAndUsers(teamID string, users []string, usersTeams map[string][]string) error
	GetUserTeams(userID string) ([]string, error)
	GetTeamUsers(teamID string) ([]string, error)
	IsTeamContainUser(teamID, userID string) (bool, error)
	DeleteTeam(teamID, userID string) error

	// Metrics management
	CleanUpOutdatedMetrics(duration time.Duration) error
	CleanUpFutureMetrics(duration time.Duration) error
	CleanupOutdatedPatternMetrics() (int64, error)
	CleanUpAbandonedRetentions() error
	RemoveMetricsByPrefix(pattern string) error
	RemoveAllMetrics() error
}

// Lock implements lock abstraction.
type Lock interface {
	Acquire(stop <-chan struct{}) (lost <-chan struct{}, error error)
	Release()
}

// Mutex implements mutex abstraction.
type Mutex interface {
	Lock() error
	Unlock() (bool, error)
	Extend() (bool, error)
}

// Logger implements logger abstraction.
type Logger interface {
	Debug() logging.EventBuilder
	Info() logging.EventBuilder
	Error() logging.EventBuilder
	Fatal() logging.EventBuilder
	Warning() logging.EventBuilder

	// Structured logging methods, use to add context fields
	String(key, value string) Logger
	Int(key string, value int) Logger
	Int64(key string, value int64) Logger
	Fields(fields map[string]interface{}) Logger

	// Get child logger with the minimum accepted level
	Level(string) (Logger, error)
	// Returns new copy of log, when need to avoid context duplication
	Clone() Logger
}

// Sender interface for implementing specified contact type sender.
type Sender interface {
	// TODO refactor: https://github.com/moira-alert/moira/issues/794
	SendEvents(events NotificationEvents, contact ContactData, trigger TriggerData, plot [][]byte, throttled bool) error
	Init(senderSettings interface{}, logger Logger, location *time.Location, dateTimeFormat string) error
}

// ImageStore is the interface for image storage providers.
type ImageStore interface {
	StoreImage(image []byte) (string, error)
	IsEnabled() bool
}

// Searcher interface implements full-text search index functionality.
type Searcher interface {
	Start() error
	Stop() error
	IsReady() bool
	SearchTriggers(options SearchOptions) (searchResults []*SearchResult, total int64, err error)
}

// PlotTheme is an interface to access plot theme styles.
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

// Clock is an interface to work with Time.
type Clock interface {
	NowUTC() time.Time
	NowUnix() int64
}
