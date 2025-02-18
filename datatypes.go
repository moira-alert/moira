package moira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/moira-alert/moira/templating"
)

const (
	// VariableContactID is used to render template with contact.ID.
	VariableContactID = "${contact_id}"
	// VariableContactValue is used to render template with contact.Value.
	VariableContactValue = "${contact_value}"
	// VariableContactType is used to render template with contact.Type.
	VariableContactType = "${contact_type}"
	// VariableTriggerID is used to render template with trigger.ID.
	VariableTriggerID = "${trigger_id}"
	// VariableTriggerName is used to render template with trigger.Name.
	VariableTriggerName = "${trigger_name}"
)

const (
	// DefaultDateTimeFormat used for formatting timestamps.
	DefaultDateTimeFormat = "15:04 02.01.2006"
	// DefaultTimeFormat used for formatting time.
	DefaultTimeFormat = "15:04"
	remindMessage     = "This metric has been in bad state for more than %v hours - please, fix."
	limit             = 1000
)

type NotificationEventSettings int

const (
	DefaultNotificationSettings NotificationEventSettings = iota
	SIFormatNumbers
)

// NotificationEvent represents trigger state changes event.
type NotificationEvent struct {
	IsTriggerEvent   bool               `json:"trigger_event,omitempty" example:"true"`
	Timestamp        int64              `json:"timestamp" example:"1590741878" format:"int64"`
	Metric           string             `json:"metric" example:"carbon.agents.*.metricsReceived"`
	Value            *float64           `json:"value,omitempty" example:"70" extensions:"x-nullable"`
	Values           map[string]float64 `json:"values,omitempty"`
	State            State              `json:"state" example:"OK"`
	TriggerID        string             `json:"trigger_id" example:"5ff37996-8927-4cab-8987-970e80d8e0a8"`
	SubscriptionID   *string            `json:"sub_id,omitempty" extensions:"x-nullable"`
	ContactID        string             `json:"contact_id,omitempty"`
	OldState         State              `json:"old_state" example:"ERROR"`
	Message          *string            `json:"msg,omitempty" extensions:"x-nullable"`
	MessageEventInfo *EventInfo         `json:"event_message" extensions:"x-nullable"`
}

// NotificationEventHistoryItem is in use to store notifications history of channel.
// (See database/redis/contact_notifications_history.go.
type NotificationEventHistoryItem struct {
	TimeStamp int64  `json:"timestamp" format:"int64"`
	Metric    string `json:"metric"`
	State     State  `json:"state"`
	OldState  State  `json:"old_state"`
	TriggerID string `json:"trigger_id"`
	ContactID string `json:"contact_id"`
}

// EventInfo - a base for creating messages.
type EventInfo struct {
	Maintenance *MaintenanceInfo `json:"maintenance,omitempty" extensions:"x-nullable"`
	Interval    *int64           `json:"interval,omitempty" example:"0" format:"int64" extensions:"x-nullable"`
}

// CreateMessage - creates a message based on EventInfo.
func (event *NotificationEvent) CreateMessage(location *time.Location) string { //nolint
	// ToDo: DEPRECATED Message in NotificationEvent
	if len(UseString(event.Message)) > 0 {
		return *event.Message
	}

	if event.MessageEventInfo == nil {
		return ""
	}

	if event.MessageEventInfo.Interval != nil && event.MessageEventInfo.Maintenance == nil {
		return fmt.Sprintf(remindMessage, *event.MessageEventInfo.Interval)
	}

	if event.MessageEventInfo.Maintenance == nil {
		return ""
	}

	messageBuffer := bytes.NewBuffer([]byte(""))
	messageBuffer.WriteString("This metric changed its state during maintenance interval.")

	if location == nil {
		location = time.UTC
	}

	if event.MessageEventInfo.Maintenance.StartUser != nil || event.MessageEventInfo.Maintenance.StartTime != nil {
		messageBuffer.WriteString(" Maintenance was set")
		if event.MessageEventInfo.Maintenance.StartUser != nil {
			messageBuffer.WriteString(" by ")
			messageBuffer.WriteString(*event.MessageEventInfo.Maintenance.StartUser)
		}
		if event.MessageEventInfo.Maintenance.StartTime != nil {
			messageBuffer.WriteString(" at ")
			messageBuffer.WriteString(time.Unix(*event.MessageEventInfo.Maintenance.StartTime, 0).In(location).Format(DefaultDateTimeFormat))
		}
		if event.MessageEventInfo.Maintenance.StopUser != nil || event.MessageEventInfo.Maintenance.StopTime != nil {
			messageBuffer.WriteString(" and removed")
			if event.MessageEventInfo.Maintenance.StopUser != nil && *event.MessageEventInfo.Maintenance.StopUser != *event.MessageEventInfo.Maintenance.StartUser {
				messageBuffer.WriteString(" by ")
				messageBuffer.WriteString(*event.MessageEventInfo.Maintenance.StopUser)
			}
			if event.MessageEventInfo.Maintenance.StopTime != nil {
				messageBuffer.WriteString(" at ")
				messageBuffer.WriteString(time.Unix(*event.MessageEventInfo.Maintenance.StopTime, 0).In(location).Format(DefaultDateTimeFormat))
			}
		}
		messageBuffer.WriteString(".")
	}
	return messageBuffer.String()
}

// NotificationEvents represents slice of NotificationEvent.
type NotificationEvents []NotificationEvent

// PopulatedDescription populates a description template using provided trigger and events data.
func (trigger *TriggerData) PopulatedDescription(events NotificationEvents) error {
	triggerDescriptionPopulater := templating.NewTriggerDescriptionPopulater(trigger.Name, events.ToTemplateEvents())
	description, err := triggerDescriptionPopulater.Populate(trigger.Desc)
	if err != nil {
		description = "Your description is using the wrong template. Since we were unable to populate your template with " +
			"data, we return it so you can parse it.\n\n" + trigger.Desc
	}

	trigger.Desc = description

	return err
}

// ToTemplateEvents converts a slice of NotificationEvent into a slice of templating.Event.
func (events NotificationEvents) ToTemplateEvents() []templating.Event {
	templateEvents := make([]templating.Event, 0, len(events))
	for _, event := range events {
		templateEvents = append(templateEvents, templating.Event{
			Metric:         event.Metric,
			MetricElements: strings.Split(event.Metric, "."),
			Timestamp:      event.Timestamp,
			State:          string(event.State),
			Value:          event.Value,
		})
	}

	return templateEvents
}

// TriggerData represents trigger object.
type TriggerData struct {
	ID            string        `json:"id" example:"292516ed-4924-4154-a62c-ebe312431fce"`
	Name          string        `json:"name" example:"Not enough disk space left"`
	Desc          string        `json:"desc" example:"check the size of /var/log"`
	Targets       []string      `json:"targets" example:"devOps.my_server.hdd.freespace_mbytes"`
	WarnValue     float64       `json:"warn_value" example:"5000"`
	ErrorValue    float64       `json:"error_value" example:"1000"`
	IsRemote      bool          `json:"is_remote" example:"false"`
	TriggerSource TriggerSource `json:"trigger_source,omitempty" example:"graphite_local"`
	ClusterId     ClusterId     `json:"cluster_id,omitempty" example:"default"`
	Tags          []string      `json:"__notifier_trigger_tags" example:"server,disk"`
}

// GetTriggerSource returns trigger source associated with the trigger.
func (trigger TriggerData) GetTriggerSource() TriggerSource {
	return trigger.TriggerSource.FillInIfNotSet(trigger.IsRemote)
}

// GetTriggerURI gets frontUri and returns triggerUrl, returns empty string on selfcheck and test notifications.
func (trigger TriggerData) GetTriggerURI(frontURI string) string {
	if trigger.ID != "" {
		return fmt.Sprintf("%s/trigger/%s", frontURI, trigger.ID)
	}
	return ""
}

// GetTags returns "[tag1][tag2]...[tagN]" string.
func (trigger *TriggerData) GetTags() string {
	var buffer bytes.Buffer
	for _, tag := range trigger.Tags {
		buffer.WriteString(fmt.Sprintf("[%s]", tag))
	}
	return buffer.String()
}

// Team is a structure that represents a group of users that share a subscriptions and contacts.
type Team struct {
	ID          string
	Name        string
	Description string
}

// ContactData represents contact object.
type ContactData struct {
	Type  string `json:"type" example:"mail"`
	Name  string `json:"name,omitempty" example:"Mail Alerts"`
	Value string `json:"value" example:"devops@example.com"`
	ID    string `json:"id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	User  string `json:"user" example:""`
	Team  string `json:"team"`
}

// ToTemplateContact converts a ContactData into a template Contact.
func (contact *ContactData) ToTemplateContact() *templating.Contact {
	return &templating.Contact{
		Type:  contact.Type,
		Value: contact.Value,
	}
}

// IDWithCount represents the number of objects for entity with given ID.
type IDWithCount struct {
	ID    string
	Count uint64
}

// ContactIDWithNotificationCount represents the number of events from notification history,
// for specified contact id.
type ContactIDWithNotificationCount IDWithCount

// SubscriptionData represents user subscription.
type SubscriptionData struct {
	Contacts          []string     `json:"contacts" example:"acd2db98-1659-4a2f-b227-52d71f6e3ba1"`
	Tags              []string     `json:"tags" example:"server,cpu"`
	Schedule          ScheduleData `json:"sched"`
	Plotting          PlottingData `json:"plotting"`
	ID                string       `json:"id" example:"292516ed-4924-4154-a62c-ebe312431fce"`
	Enabled           bool         `json:"enabled" example:"true"`
	AnyTags           bool         `json:"any_tags" example:"false"`
	IgnoreWarnings    bool         `json:"ignore_warnings,omitempty" example:"false"`
	IgnoreRecoverings bool         `json:"ignore_recoverings,omitempty" example:"false"`
	ThrottlingEnabled bool         `json:"throttling" example:"false"`
	User              string       `json:"user" example:""`
	TeamID            string       `json:"team_id" example:"324516ed-4924-4154-a62c-eb124234fce"`
}

// PlottingData represents plotting settings.
type PlottingData struct {
	Enabled bool   `json:"enabled" example:"true"`
	Theme   string `json:"theme" example:"dark"`
}

// ScheduleData represents subscription schedule.
type ScheduleData struct {
	Days           []ScheduleDataDay `json:"days" validate:"dive"`
	TimezoneOffset int64             `json:"tzOffset" example:"-60" format:"int64"`
	StartOffset    int64             `json:"startOffset" example:"0" format:"int64"`
	EndOffset      int64             `json:"endOffset" example:"1439" format:"int64"`
}

// ScheduleDataDay represents week day of schedule.
type ScheduleDataDay struct {
	Enabled bool    `json:"enabled" example:"true"`
	Name    DayName `json:"name,omitempty" example:"Mon" swaggertype:"string" validate:"oneof=Mon Tue Wed Thu Fri Sat Sun"`
}

// DayName represents the day name used in ScheduleDataDay.
type DayName string

// Constants for day names.
const (
	Monday    DayName = "Mon"
	Tuesday   DayName = "Tue"
	Wednesday DayName = "Wed"
	Thursday  DayName = "Thu"
	Friday    DayName = "Fri"
	Saturday  DayName = "Sat"
	Sunday    DayName = "Sun"
)

// DaysOrder represents the order of days in week.
var DaysOrder = [...]DayName{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}

// GetFilledScheduleDataDays returns slice of ScheduleDataDay with ScheduleDataDay.Enabled field set from param.
// Days are ordered with DaysOrder.
func GetFilledScheduleDataDays(enabled bool) []ScheduleDataDay {
	days := make([]ScheduleDataDay, 0, len(DaysOrder))

	for _, d := range DaysOrder {
		days = append(days, ScheduleDataDay{
			Name:    d,
			Enabled: enabled,
		})
	}

	return days
}

const (
	// DefaultTimezoneOffset is a default value for timezone offset for (GMT+3) used in NewDefaultScheduleData.
	DefaultTimezoneOffset = -180
	// DefaultStartOffset is a default value for start offset for (GMT+3) used in NewDefaultScheduleData.
	DefaultStartOffset = 0
	// DefaultEndOffset is a default value for end offset for (GMT+3) used in NewDefaultScheduleData.
	DefaultEndOffset = 1439
)

// NewDefaultScheduleData returns the default ScheduleData which can be used in Trigger.
func NewDefaultScheduleData() *ScheduleData {
	return &ScheduleData{
		Days:           GetFilledScheduleDataDays(true),
		TimezoneOffset: DefaultTimezoneOffset,
		StartOffset:    DefaultStartOffset,
		EndOffset:      DefaultEndOffset,
	}
}

// ScheduledNotification represent notification object.
type ScheduledNotification struct {
	Event     NotificationEvent `json:"event"`
	Trigger   TriggerData       `json:"trigger"`
	Contact   ContactData       `json:"contact"`
	Plotting  PlottingData      `json:"plotting"`
	Throttled bool              `json:"throttled" example:"false"`
	SendFail  int               `json:"send_fail" example:"0"`
	Timestamp int64             `json:"timestamp" example:"1594471927" format:"int64"`
	CreatedAt int64             `json:"created_at,omitempty" example:"1594471900" format:"int64"`
}

type scheduledNotificationState int

const (
	ResavedNotification scheduledNotificationState = iota
	ValidNotification
	RemovedNotification
)

// Less is needed for the ScheduledNotification to match the Comparable interface.
func (notification *ScheduledNotification) Less(other Comparable) (bool, error) {
	otherNotification, ok := other.(*ScheduledNotification)
	if !ok {
		return false, fmt.Errorf("cannot to compare ScheduledNotification with different type")
	}

	return notification.Timestamp < otherNotification.Timestamp, nil
}

// IsDelayed checks if the notification is delayed, the difference between the send time and the creation time
// is greater than the delayedTime.
func (notification *ScheduledNotification) IsDelayed(delayedTime int64) bool {
	return notification.CreatedAt != 0 && notification.Timestamp-notification.CreatedAt > delayedTime
}

/*
GetState checks:
  - If the trigger for which the notification was generated has been deleted, returns Removed state.
  - If the metric is on Maintenance, returns Resaved state.
  - If the trigger is on Maintenance, returns Resaved state.

Otherwise returns Valid state.
*/
func (notification *ScheduledNotification) GetState(triggerCheck *CheckData) scheduledNotificationState {
	if triggerCheck == nil {
		return RemovedNotification
	}

	if triggerCheck.IsMetricOnMaintenance(notification.Event.Metric) || triggerCheck.IsTriggerOnMaintenance() {
		return ResavedNotification
	}

	return ValidNotification
}

// MatchedMetric represents parsed and matched metric data.
type MatchedMetric struct {
	Metric             string
	Patterns           []string
	Value              float64
	Timestamp          int64
	RetentionTimestamp int64
	Retention          int
}

// MetricValue represents metric data.
type MetricValue struct {
	RetentionTimestamp int64   `json:"step,omitempty" format:"int64"`
	Timestamp          int64   `json:"ts" format:"int64"`
	Value              float64 `json:"value"`
}

const (
	// FallingTrigger represents falling trigger type, in which OK > WARN > ERROR.
	FallingTrigger = "falling"
	// RisingTrigger represents rising trigger type, in which OK < WARN < ERROR.
	RisingTrigger = "rising"
	// ExpressionTrigger represents trigger type with custom user expression.
	ExpressionTrigger = "expression"
)

// Trigger represents trigger data object.
type Trigger struct {
	ID               string          `json:"id" example:"292516ed-4924-4154-a62c-ebe312431fce"`
	Name             string          `json:"name" example:"Not enough disk space left"`
	Desc             *string         `json:"desc,omitempty" example:"check the size of /var/log" extensions:"x-nullable"`
	Targets          []string        `json:"targets" example:"devOps.my_server.hdd.freespace_mbytes"`
	WarnValue        *float64        `json:"warn_value" example:"5000" extensions:"x-nullable"`
	ErrorValue       *float64        `json:"error_value" example:"1000" extensions:"x-nullable"`
	TriggerType      string          `json:"trigger_type" example:"rising"`
	Tags             []string        `json:"tags" example:"server,disk"`
	TTLState         *TTLState       `json:"ttl_state,omitempty" example:"NODATA" extensions:"x-nullable"`
	TTL              int64           `json:"ttl,omitempty" example:"600" format:"int64"`
	Schedule         *ScheduleData   `json:"sched,omitempty" extensions:"x-nullable"`
	Expression       *string         `json:"expression,omitempty" example:"" extensions:"x-nullable"`
	PythonExpression *string         `json:"python_expression,omitempty" extensions:"x-nullable"`
	Patterns         []string        `json:"patterns" example:""`
	TriggerSource    TriggerSource   `json:"trigger_source,omitempty" example:"graphite_local"`
	ClusterId        ClusterId       `json:"cluster_id,omitempty" example:"default"`
	MuteNewMetrics   bool            `json:"mute_new_metrics" example:"false"`
	AloneMetrics     map[string]bool `json:"alone_metrics" example:"t1:true"`
	CreatedAt        *int64          `json:"created_at" format:"int64" extensions:"x-nullable"`
	UpdatedAt        *int64          `json:"updated_at" format:"int64" extensions:"x-nullable"`
	CreatedBy        string          `json:"created_by"`
	UpdatedBy        string          `json:"updated_by"`
}

const (
	// DefaultTTL is a default value for Trigger.TTL.
	DefaultTTL = 600
)

// ClusterKey returns cluster key composed of trigger source and cluster id associated with the trigger.
func (trigger *Trigger) ClusterKey() ClusterKey {
	return MakeClusterKey(trigger.TriggerSource, trigger.ClusterId)
}

// TriggerSource is a enum which values correspond to types of moira's metric sources.
type TriggerSource string

var (
	TriggerSourceNotSet TriggerSource = ""
	GraphiteLocal       TriggerSource = "graphite_local"
	GraphiteRemote      TriggerSource = "graphite_remote"
	PrometheusRemote    TriggerSource = "prometheus_remote"
)

func (s *TriggerSource) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	source := TriggerSource(v)
	if source != GraphiteLocal && source != GraphiteRemote && source != PrometheusRemote {
		*s = TriggerSourceNotSet
		return nil
	}

	*s = source
	return nil
}

// Needed for backwards compatibility with moira versions that used only isRemote flag.
func (triggerSource TriggerSource) FillInIfNotSet(isRemote bool) TriggerSource {
	if triggerSource == TriggerSourceNotSet {
		if isRemote {
			return GraphiteRemote
		} else {
			return GraphiteLocal
		}
	}
	return triggerSource
}

func (triggerSource TriggerSource) String() string {
	return string(triggerSource)
}

// ClusterId represent the unique id for each cluster with the same TriggerSource.
type ClusterId string

var (
	ClusterNotSet  ClusterId = ""
	DefaultCluster ClusterId = "default"
)

// FillInIfNotSet returns new ClusterId with value set to default if it was empty.
func (clusterId ClusterId) FillInIfNotSet() ClusterId {
	if clusterId == ClusterNotSet {
		return DefaultCluster
	}
	return clusterId
}

func (clusterId ClusterId) String() string {
	return string(clusterId)
}

// ClusterKey represents unique key of a metric source.
type ClusterKey struct {
	TriggerSource TriggerSource
	ClusterId     ClusterId
}

var (
	DefaultLocalCluster            = MakeClusterKey(GraphiteLocal, DefaultCluster)
	DefaultGraphiteRemoteCluster   = MakeClusterKey(GraphiteRemote, DefaultCluster)
	DefaultPrometheusRemoteCluster = MakeClusterKey(PrometheusRemote, DefaultCluster)
)

// MakeClusterKey creates new cluster key with given trigger source and cluster id.
func MakeClusterKey(triggerSource TriggerSource, clusterId ClusterId) ClusterKey {
	return ClusterKey{
		TriggerSource: triggerSource,
		ClusterId:     clusterId,
	}
}

func (clusterKey ClusterKey) String() string {
	return fmt.Sprintf("%s.%s", clusterKey.TriggerSource, clusterKey.ClusterId)
}

// TriggerCheck represents trigger data with last check data and check timestamp.
type TriggerCheck struct {
	Trigger
	Throttling int64             `json:"throttling" example:"0" format:"int64"`
	LastCheck  CheckData         `json:"last_check"`
	Highlights map[string]string `json:"highlights"`
}

// SearchOptions represents the options that can be selected when searching triggers.
type SearchOptions struct {
	Page                  int64
	Size                  int64
	OnlyProblems          bool
	SearchString          string
	Tags                  []string
	CreatedBy             string
	NeedSearchByCreatedBy bool
	CreatePager           bool
	PagerID               string
	PagerTTL              time.Duration
}

// MaintenanceCheck set maintenance user, time.
type MaintenanceCheck interface {
	SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64)
	GetMaintenance() (MaintenanceInfo, int64)
}

// CheckData represents last trigger check data.
type CheckData struct {
	Metrics map[string]MetricState `json:"metrics"`
	// MetricsToTargetRelation is a map that holds relation between metric names that was alone during last
	// check and targets that fetched this metric
	//	{"t1": "metric.name.1", "t2": "metric.name.2"}
	MetricsToTargetRelation map[string]string `json:"metrics_to_target_relation" example:"t1:metric.name.1,t2:metric.name.2"`
	Score                   int64             `json:"score" example:"100" format:"int64"`
	State                   State             `json:"state" example:"OK"`
	Maintenance             int64             `json:"maintenance,omitempty" example:"0" format:"int64"`
	MaintenanceInfo         MaintenanceInfo   `json:"maintenance_info"`
	// Timestamp - time, which means when the checker last checked this trigger, this value stops updating if the trigger does not receive metrics
	Timestamp      int64 `json:"timestamp,omitempty" example:"1590741916" format:"int64"`
	EventTimestamp int64 `json:"event_timestamp,omitempty" example:"1590741878" format:"int64"`
	// LastSuccessfulCheckTimestamp - time of the last check of the trigger, during which there were no errors
	LastSuccessfulCheckTimestamp int64  `json:"last_successful_check_timestamp" example:"1590741916" format:"int64"`
	Suppressed                   bool   `json:"suppressed,omitempty" example:"true"`
	SuppressedState              State  `json:"suppressed_state,omitempty"`
	Message                      string `json:"msg,omitempty"`
	Clock                        Clock  `json:"-"`
}

// Need to not show the user metrics that should have been deleted due to ttlState = Del,
// but remained in the database because their Maintenance did not expire.
func (checkData *CheckData) RemoveDeadMetrics() {
	for metricName, metricState := range checkData.Metrics {
		if metricState.DeletedButKept {
			delete(checkData.Metrics, metricName)
		}
	}
}

// RemoveMetricState is a function that removes MetricState from map of states.
func (checkData CheckData) RemoveMetricState(metricName string) {
	delete(checkData.Metrics, metricName)
}

// RemoveMetricsToTargetRelation is a function that sets an empty map to MetricsToTargetRelation.
func (checkData *CheckData) RemoveMetricsToTargetRelation() {
	checkData.MetricsToTargetRelation = make(map[string]string)
}

// IsTriggerOnMaintenance checks if the trigger is on Maintenance.
func (checkData *CheckData) IsTriggerOnMaintenance() bool {
	return checkData.Clock.NowUnix() <= checkData.Maintenance
}

// IsMetricOnMaintenance checks if the metric of the given trigger is on Maintenance.
func (checkData *CheckData) IsMetricOnMaintenance(metric string) bool {
	if checkData.Metrics == nil {
		return false
	}

	metricState, ok := checkData.Metrics[metric]
	if !ok {
		return false
	}

	return checkData.Clock.NowUnix() <= metricState.Maintenance
}

// MetricState represents metric state data for given timestamp.
type MetricState struct {
	EventTimestamp  int64              `json:"event_timestamp" example:"1590741878" format:"int64"`
	State           State              `json:"state" example:"OK"`
	Suppressed      bool               `json:"suppressed" example:"false"`
	SuppressedState State              `json:"suppressed_state,omitempty"`
	Timestamp       int64              `json:"timestamp" example:"1590741878" format:"int64"`
	Value           *float64           `json:"value,omitempty" example:"70" extensions:"x-nullable"`
	Values          map[string]float64 `json:"values,omitempty"`
	Maintenance     int64              `json:"maintenance,omitempty" example:"0" format:"int64"`
	MaintenanceInfo MaintenanceInfo    `json:"maintenance_info"`
	// DeletedButKept controls whether the metric is shown to the user if the trigger has ttlState = Del
	// and the metric is in Maintenance. The metric remains in the database
	DeletedButKept bool `json:"deleted_but_kept,omitempty" example:"false"`
	// AloneMetrics    map[string]string  `json:"alone_metrics"` // represents a relation between name of alone metrics and their targets
}

// SetMaintenance set maintenance user, time for MetricState.
func (metricState *MetricState) SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64) {
	metricState.MaintenanceInfo = *maintenanceInfo
	metricState.Maintenance = maintenance
}

// GetMaintenance return metricState MaintenanceInfo.
func (metricState *MetricState) GetMaintenance() (MaintenanceInfo, int64) {
	return metricState.MaintenanceInfo, metricState.Maintenance
}

// MaintenanceInfo represents user and time set/unset maintenance.
type MaintenanceInfo struct {
	StartUser *string `json:"setup_user" extensions:"x-nullable"`
	StartTime *int64  `json:"setup_time" example:"0" format:"int64" extensions:"x-nullable"`
	StopUser  *string `json:"remove_user" extensions:"x-nullable"`
	StopTime  *int64  `json:"remove_time" example:"0" format:"int64" extensions:"x-nullable"`
}

// Set maintanace start and stop users and times.
func (maintenanceInfo *MaintenanceInfo) Set(startUser *string, startTime *int64, stopUser *string, stopTime *int64) {
	maintenanceInfo.StartUser = startUser
	maintenanceInfo.StartTime = startTime
	maintenanceInfo.StopUser = stopUser
	maintenanceInfo.StopTime = stopTime
}

// MetricEvent represents filter metric event.
type MetricEvent struct {
	Metric  string `json:"metric"`
	Pattern string `json:"pattern"`
}

// SubscribeMetricEventsParams represents params of subscription.
type SubscribeMetricEventsParams struct {
	BatchSize int64
	Delay     time.Duration
}

// SearchHighlight represents highlight.
type SearchHighlight struct {
	Field string
	Value string
}

// SearchResult represents fulltext search result.
type SearchResult struct {
	ObjectID   string
	Highlights []SearchHighlight
}

// GetSubjectState returns the most critical state of events.
func (events NotificationEvents) getSubjectState() State {
	result := StateOK
	states := make(map[State]bool)
	for _, event := range events {
		states[event.State] = true
	}
	for _, state := range eventStatesPriority {
		if states[state] {
			result = state
		}
	}
	return result
}

// GetLastState returns the last state of events.
func (events NotificationEvents) getLastState() State {
	if len(events) != 0 {
		return events[len(events)-1].State
	}
	return StateNODATA
}

// Returns the current state depending on the throttled parameter.
func (events NotificationEvents) GetCurrentState(throttled bool) State {
	if throttled {
		return events.getLastState()
	}
	return events.getSubjectState()
}

// GetKey return notification key to prevent duplication to the same contact.
func (notification *ScheduledNotification) GetKey() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%d:%s:%d:%t:%d",
		notification.Contact.Type,
		notification.Contact.Value,
		notification.Event.TriggerID,
		notification.Event.Metric,
		notification.Event.State,
		notification.Event.Timestamp,
		notification.Event.GetMetricsValues(DefaultNotificationSettings),
		notification.SendFail,
		notification.Throttled,
		notification.Timestamp,
	)
}

// IsScheduleAllows check if the time is in the allowed schedule interval.
func (schedule *ScheduleData) IsScheduleAllows(ts int64) bool {
	if schedule == nil {
		return true
	}
	endOffset, startOffset := schedule.EndOffset, schedule.StartOffset
	if schedule.EndOffset < schedule.StartOffset {
		endOffset = schedule.EndOffset + 24*60 //nolint
	}
	timestamp := ts - ts%60 - schedule.TimezoneOffset*60 //nolint
	date := time.Unix(timestamp, 0).UTC()
	if !schedule.Days[int(date.Weekday()+6)%7].Enabled { //nolint
		return false
	}
	dayStart := time.Unix(timestamp-timestamp%(24*3600), 0).UTC()
	startDayTime := dayStart.Add(time.Duration(startOffset) * time.Minute)
	endDayTime := dayStart.Add(time.Duration(endOffset) * time.Minute)
	if endOffset < 24*60 {
		if (date.After(startDayTime) || date.Equal(startDayTime)) && (date.Before(endDayTime) || date.Equal(endDayTime)) {
			return true
		}
	} else {
		endDayTime = endDayTime.Add(-time.Hour * 24) //nolint
		if date.Before(endDayTime) || date.After(startDayTime) {
			return true
		}
	}
	return false
}

func (event NotificationEvent) String() string {
	return fmt.Sprintf("TriggerId: %s, Metric: %s, Values: %s, OldState: %s, State: %s, Message: '%s', Timestamp: %v", event.TriggerID, event.Metric, event.GetMetricsValues(DefaultNotificationSettings), event.OldState, event.State, event.CreateMessage(nil), event.Timestamp)
}

// GetMetricsValues gets event metric value and format it to human readable presentation.
func (event NotificationEvent) GetMetricsValues(settings NotificationEventSettings) string {
	targetNames := make([]string, 0, len(event.Values))
	for targetName := range event.Values {
		targetNames = append(targetNames, targetName)
	}

	if len(targetNames) == 0 {
		return "—"
	}

	if len(targetNames) == 1 {
		switch settings {
		case SIFormatNumbers:
			if event.Values[targetNames[0]] >= limit {
				return humanize.SIWithDigits(event.Values[targetNames[0]], 3, "")
			}
			return humanize.FtoaWithDigits(event.Values[targetNames[0]], 3)
		}
		return strconv.FormatFloat(event.Values[targetNames[0]], 'f', -1, 64)
	}

	var builder strings.Builder
	sort.Strings(targetNames)
	for i, targetName := range targetNames {
		builder.WriteString(targetName)
		builder.WriteString(": ")
		value := strconv.FormatFloat(event.Values[targetName], 'f', -1, 64)
		switch settings {
		case SIFormatNumbers:
			if event.Values[targetName] >= limit {
				value = humanize.SIWithDigits(event.Values[targetName], 3, "")
			} else {
				value = humanize.FtoaWithDigits(event.Values[targetName], 3)
			}
		}
		builder.WriteString(value)
		if i < len(targetNames)-1 {
			builder.WriteString(", ")
		}
	}

	return builder.String()
}

// FormatTimestamp gets event timestamp and format it using given location to human readable presentation.
func (event NotificationEvent) FormatTimestamp(location *time.Location, timeFormat string) string {
	timestamp := time.Unix(event.Timestamp, 0).In(location)
	formattedTime := timestamp.Format(timeFormat)
	offset := timestamp.Format("-07:00")

	return formattedTime + " (GMT" + offset + ")"
}

// GetOrCreateMetricState gets metric state from check data or create new if CheckData has no state for given metric.
func (checkData *CheckData) GetOrCreateMetricState(metric string, muteFirstMetric bool, checkPointGap int64) MetricState {
	if _, ok := checkData.Metrics[metric]; !ok {
		checkData.Metrics[metric] = createEmptyMetricState(muteFirstMetric, checkPointGap, checkData.Clock)
	}

	return checkData.Metrics[metric]
}

// SetMaintenance set maintenance user, time for CheckData.
func (checkData *CheckData) SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64) {
	checkData.MaintenanceInfo = *maintenanceInfo
	checkData.Maintenance = maintenance
}

// GetMaintenance return metricState MaintenanceInfo.
func (checkData *CheckData) GetMaintenance() (MaintenanceInfo, int64) {
	return checkData.MaintenanceInfo, checkData.Maintenance
}

func createEmptyMetricState(muteFirstMetric bool, checkPointGap int64, clock Clock) MetricState {
	metric := MetricState{
		Timestamp:      clock.NowUnix(),
		EventTimestamp: clock.NowUnix() - checkPointGap,
	}

	if muteFirstMetric {
		metric.State = StateOK
	} else {
		metric.State = StateNODATA
	}

	return metric
}

// GetCheckPoint gets check point for given MetricState.
// CheckPoint is the timestamp from which to start checking the current state of the metric.
func (metricState *MetricState) GetCheckPoint(checkPointGap int64) int64 {
	return int64(math.Max(float64(metricState.Timestamp-checkPointGap), float64(metricState.EventTimestamp)))
}

// GetEventTimestamp gets event timestamp for given metric.
func (metricState MetricState) GetEventTimestamp() int64 {
	if metricState.EventTimestamp == 0 {
		return metricState.Timestamp
	}
	return metricState.EventTimestamp
}

// GetEventTimestamp gets event timestamp for given check.
func (checkData CheckData) GetEventTimestamp() int64 {
	if checkData.EventTimestamp == 0 {
		return checkData.Timestamp
	}
	return checkData.EventTimestamp
}

// IsSimple checks triggers patterns.
// If patterns more than one or it contains standard graphite wildcard symbols,
// when this target can contain more then one metrics, and is it not simple trigger.
func (trigger *Trigger) IsSimple() bool {
	if len(trigger.Targets) > 1 || len(trigger.Patterns) > 1 {
		return false
	}
	for _, pattern := range trigger.Patterns {
		if strings.ContainsAny(pattern, "*{?[") || strings.Contains(pattern, "seriesByTag") {
			return false
		}
	}
	return true
}

// UpdateScore update and return checkData score, based on metric states and checkData state.
func (checkData *CheckData) UpdateScore() int64 {
	checkData.Score = stateScores[checkData.State]
	for _, metricData := range checkData.Metrics {
		checkData.Score += stateScores[metricData.State]
	}
	return checkData.Score
}

// MustIgnore returns true if given state transition must be ignored.
func (subscription *SubscriptionData) MustIgnore(eventData *NotificationEvent) bool {
	if oldStateWeight, ok := eventStateWeight[eventData.OldState]; ok {
		if newStateWeight, ok := eventStateWeight[eventData.State]; ok {
			delta := newStateWeight - oldStateWeight
			if delta < 0 {
				if delta == -1 && (subscription.IgnoreRecoverings || subscription.IgnoreWarnings) {
					return true
				}
				return subscription.IgnoreRecoverings
			}
			if delta == 1 {
				return subscription.IgnoreWarnings
			}
		}
	}
	return false
}

// isAnonymous checks if user is Anonymous or empty.
func isAnonymous(user string) bool {
	return user == "anonymous" || user == ""
}

// SetMaintenanceUserAndTime set startuser and starttime or stopuser and stoptime for MaintenanceInfo.
func SetMaintenanceUserAndTime(maintenanceCheck MaintenanceCheck, maintenance int64, user string, callMaintenance int64) {
	maintenanceInfo, _ := maintenanceCheck.GetMaintenance()
	if maintenance < callMaintenance {
		if (maintenanceInfo.StartUser != nil && !isAnonymous(*maintenanceInfo.StartUser)) || !isAnonymous(user) {
			maintenanceInfo.StopUser = &user
			maintenanceInfo.StopTime = &callMaintenance
		}
		if isAnonymous(user) {
			maintenanceInfo.StopUser = nil
			maintenanceInfo.StopTime = nil
		}
	} else {
		if !isAnonymous(user) {
			maintenanceInfo.Set(&user, &callMaintenance, nil, nil)
		} else {
			maintenanceInfo.Set(nil, nil, nil, nil)
		}
	}
	maintenanceCheck.SetMaintenance(&maintenanceInfo, maintenance)
}

// SchedulerParams is the parameters for notifier.Scheduler essential for scheduling notification.
type SchedulerParams struct {
	Event        NotificationEvent
	Trigger      TriggerData
	Contact      ContactData
	Plotting     PlottingData
	ThrottledOld bool
	// SendFail is amount of failed send attempts
	SendFail int
}
