package moira

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira/templating"
)

const (
	// VariableContactID is used to render template with contact.ID
	VariableContactID = "${contact_id}"
	// VariableContactValue is used to render template with contact.Value
	VariableContactValue = "${contact_value}"
	// VariableContactType is used to render template with contact.Type
	VariableContactType = "${contact_type}"
	// VariableTriggerID is used to render template with trigger.ID
	VariableTriggerID = "${trigger_id}"
	// VariableTriggerName is used to render template with trigger.Name
	VariableTriggerName = "${trigger_name}"
)

const (
	format        = "15:04 02.01.2006"
	remindMessage = "This metric has been in bad state for more than %v hours - please, fix."
)

// NotificationEvent represents trigger state changes event
type NotificationEvent struct {
	IsTriggerEvent   bool               `json:"trigger_event,omitempty"`
	Timestamp        int64              `json:"timestamp"`
	Metric           string             `json:"metric"`
	Value            *float64           `json:"value,omitempty"`
	Values           map[string]float64 `json:"values,omitempty"`
	State            State              `json:"state"`
	TriggerID        string             `json:"trigger_id"`
	SubscriptionID   *string            `json:"sub_id,omitempty"`
	ContactID        string             `json:"contactId,omitempty"`
	OldState         State              `json:"old_state"`
	Message          *string            `json:"msg,omitempty"`
	MessageEventInfo *EventInfo         `json:"event_message"`
}

// EventInfo - a base for creating messages.
type EventInfo struct {
	Maintenance *MaintenanceInfo `json:"maintenance,omitempty"`
	Interval    *int64           `json:"interval,omitempty"`
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
			messageBuffer.WriteString(time.Unix(*event.MessageEventInfo.Maintenance.StartTime, 0).In(location).Format(format))
		}
		if event.MessageEventInfo.Maintenance.StopUser != nil || event.MessageEventInfo.Maintenance.StopTime != nil {
			messageBuffer.WriteString(" and removed")
			if event.MessageEventInfo.Maintenance.StopUser != nil && *event.MessageEventInfo.Maintenance.StopUser != *event.MessageEventInfo.Maintenance.StartUser {
				messageBuffer.WriteString(" by ")
				messageBuffer.WriteString(*event.MessageEventInfo.Maintenance.StopUser)
			}
			if event.MessageEventInfo.Maintenance.StopTime != nil {
				messageBuffer.WriteString(" at ")
				messageBuffer.WriteString(time.Unix(*event.MessageEventInfo.Maintenance.StopTime, 0).In(location).Format(format))
			}
		}
		messageBuffer.WriteString(".")
	}
	return messageBuffer.String()
}

// NotificationEvents represents slice of NotificationEvent
type NotificationEvents []NotificationEvent

func (trigger *TriggerData) PopulatedDescription(events NotificationEvents) error {
	description, err := templating.Populate(trigger.Name, trigger.Desc, NotificationEventsToTemplatingEvents(events))
	if err != nil {
		description = "Your description is using the wrong template. Since we were unable to populate your template with " +
			"data, we return it so you can parse it.\n\n" + trigger.Desc
	}

	trigger.Desc = description

	return err
}

func NotificationEventsToTemplatingEvents(events NotificationEvents) []templating.Event {
	templatingEvents := make([]templating.Event, 0, len(events))
	for _, event := range events {
		templatingEvents = append(templatingEvents, templating.Event{
			Metric:         event.Metric,
			MetricElements: strings.Split(event.Metric, "."),
			Timestamp:      event.Timestamp,
			State:          string(event.State),
			Value:          event.Value,
		})
	}

	return templatingEvents
}

// TriggerData represents trigger object
type TriggerData struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Targets    []string `json:"targets"`
	WarnValue  float64  `json:"warn_value"`
	ErrorValue float64  `json:"error_value"`
	IsRemote   bool     `json:"is_remote"`
	Tags       []string `json:"__notifier_trigger_tags"`
}

// GetTriggerURI gets frontUri and returns triggerUrl, returns empty string on selfcheck and test notifications
func (trigger TriggerData) GetTriggerURI(frontURI string) string {
	if trigger.ID != "" {
		return fmt.Sprintf("%s/trigger/%s", frontURI, trigger.ID)
	}
	return ""
}

// ContactData represents contact object
type ContactData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id"`
	User  string `json:"user"`
}

// SubscriptionData represents user subscription
type SubscriptionData struct {
	Contacts          []string     `json:"contacts"`
	Tags              []string     `json:"tags"`
	Schedule          ScheduleData `json:"sched"`
	Plotting          PlottingData `json:"plotting"`
	ID                string       `json:"id"`
	Enabled           bool         `json:"enabled"`
	AnyTags           bool         `json:"any_tags"`
	IgnoreWarnings    bool         `json:"ignore_warnings,omitempty"`
	IgnoreRecoverings bool         `json:"ignore_recoverings,omitempty"`
	ThrottlingEnabled bool         `json:"throttling"`
	User              string       `json:"user"`
}

// PlottingData represents plotting settings
type PlottingData struct {
	Enabled bool   `json:"enabled"`
	Theme   string `json:"theme"`
}

// ScheduleData represents subscription schedule
type ScheduleData struct {
	Days           []ScheduleDataDay `json:"days"`
	TimezoneOffset int64             `json:"tzOffset"`
	StartOffset    int64             `json:"startOffset"`
	EndOffset      int64             `json:"endOffset"`
}

// ScheduleDataDay represents week day of schedule
type ScheduleDataDay struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name,omitempty"`
}

// ScheduledNotification represent notification object
type ScheduledNotification struct {
	Event     NotificationEvent `json:"event"`
	Trigger   TriggerData       `json:"trigger"`
	Contact   ContactData       `json:"contact"`
	Plotting  PlottingData      `json:"plotting"`
	Throttled bool              `json:"throttled"`
	SendFail  int               `json:"send_fail"`
	Timestamp int64             `json:"timestamp"`
}

// MatchedMetric represents parsed and matched metric data
type MatchedMetric struct {
	Metric             string
	Patterns           []string
	Value              float64
	Timestamp          int64
	RetentionTimestamp int64
	Retention          int
}

// MetricValue represents metric data
type MetricValue struct {
	RetentionTimestamp int64   `json:"step,omitempty"`
	Timestamp          int64   `json:"ts"`
	Value              float64 `json:"value"`
}

const (
	// FallingTrigger represents falling trigger type, in which OK > WARN > ERROR
	FallingTrigger = "falling"
	// RisingTrigger represents rising trigger type, in which OK < WARN < ERROR
	RisingTrigger = "rising"
	// ExpressionTrigger represents trigger type with custom user expression
	ExpressionTrigger = "expression"
)

// Trigger represents trigger data object
type Trigger struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Desc             *string         `json:"desc,omitempty"`
	Targets          []string        `json:"targets"`
	WarnValue        *float64        `json:"warn_value"`
	ErrorValue       *float64        `json:"error_value"`
	TriggerType      string          `json:"trigger_type"`
	Tags             []string        `json:"tags"`
	TTLState         *TTLState       `json:"ttl_state,omitempty"`
	TTL              int64           `json:"ttl,omitempty"`
	Schedule         *ScheduleData   `json:"sched,omitempty"`
	Expression       *string         `json:"expression,omitempty"`
	PythonExpression *string         `json:"python_expression,omitempty"`
	Patterns         []string        `json:"patterns"`
	IsRemote         bool            `json:"is_remote"`
	MuteNewMetrics   bool            `json:"mute_new_metrics"`
	AloneMetrics     map[string]bool `json:"alone_metrics"`
}

// TriggerCheck represents trigger data with last check data and check timestamp
type TriggerCheck struct {
	Trigger
	Throttling int64             `json:"throttling"`
	LastCheck  CheckData         `json:"last_check"`
	Highlights map[string]string `json:"highlights"`
}

// MaintenanceCheck set maintenance user, time
type MaintenanceCheck interface {
	SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64)
	GetMaintenance() (MaintenanceInfo, int64)
}

// CheckData represents last trigger check data
type CheckData struct {
	Metrics map[string]MetricState `json:"metrics"`
	// MetricsToTargetRelation is a map that holds relation between metric names that was alone during last
	// check and targets that fetched this metric
	MetricsToTargetRelation      map[string]string `json:"metrics_to_target_relation"`
	Score                        int64             `json:"score"`
	State                        State             `json:"state"`
	Maintenance                  int64             `json:"maintenance,omitempty"`
	MaintenanceInfo              MaintenanceInfo   `json:"maintenance_info"`
	Timestamp                    int64             `json:"timestamp,omitempty"`
	EventTimestamp               int64             `json:"event_timestamp,omitempty"`
	LastSuccessfulCheckTimestamp int64             `json:"last_successful_check_timestamp"`
	Suppressed                   bool              `json:"suppressed,omitempty"`
	SuppressedState              State             `json:"suppressed_state,omitempty"`
	Message                      string            `json:"msg,omitempty"`
}

// RemoveMetricState is a function that removes MetricState from map of states.
func (checkData CheckData) RemoveMetricState(metricName string) {
	delete(checkData.Metrics, metricName)
}

// RemoveMetricsToTargetRelation is a function that sets an empty map to MetricsToTargetRelation.
func (checkData *CheckData) RemoveMetricsToTargetRelation() {
	checkData.MetricsToTargetRelation = make(map[string]string)
}

// MetricState represents metric state data for given timestamp
type MetricState struct {
	EventTimestamp  int64              `json:"event_timestamp"`
	State           State              `json:"state"`
	Suppressed      bool               `json:"suppressed"`
	SuppressedState State              `json:"suppressed_state,omitempty"`
	Timestamp       int64              `json:"timestamp"`
	Value           *float64           `json:"value,omitempty"`
	Values          map[string]float64 `json:"values,omitempty"`
	Maintenance     int64              `json:"maintenance,omitempty"`
	MaintenanceInfo MaintenanceInfo    `json:"maintenance_info"`
	// AloneMetrics    map[string]string  `json:"alone_metrics"` // represents a relation between name of alone metrics and their targets
}

// SetMaintenance set maintenance user, time for MetricState
func (metricState *MetricState) SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64) {
	metricState.MaintenanceInfo = *maintenanceInfo
	metricState.Maintenance = maintenance
}

// GetMaintenance return metricState MaintenanceInfo
func (metricState *MetricState) GetMaintenance() (MaintenanceInfo, int64) {
	return metricState.MaintenanceInfo, metricState.Maintenance
}

// MaintenanceInfo represents user and time set/unset maintenance
type MaintenanceInfo struct {
	StartUser *string `json:"setup_user"`
	StartTime *int64  `json:"setup_time"`
	StopUser  *string `json:"remove_user"`
	StopTime  *int64  `json:"remove_time"`
}

// Set maintanace start and stop users and times
func (maintenanceInfo *MaintenanceInfo) Set(startUser *string, startTime *int64, stopUser *string, stopTime *int64) {
	maintenanceInfo.StartUser = startUser
	maintenanceInfo.StartTime = startTime
	maintenanceInfo.StopUser = stopUser
	maintenanceInfo.StopTime = stopTime
}

// MetricEvent represents filter metric event
type MetricEvent struct {
	Metric  string `json:"metric"`
	Pattern string `json:"pattern"`
}

// SearchHighlight represents highlight
type SearchHighlight struct {
	Field string
	Value string
}

// SearchResult represents fulltext search result
type SearchResult struct {
	ObjectID   string
	Highlights []SearchHighlight
}

// GetSubjectState returns the most critical state of events
func (events NotificationEvents) GetSubjectState() State {
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

// GetTags returns "[tag1][tag2]...[tagN]" string
func (trigger *TriggerData) GetTags() string {
	var buffer bytes.Buffer
	for _, tag := range trigger.Tags {
		buffer.WriteString(fmt.Sprintf("[%s]", tag))
	}
	return buffer.String()
}

// GetKey return notification key to prevent duplication to the same contact
func (notification *ScheduledNotification) GetKey() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%d:%s:%d:%t:%d",
		notification.Contact.Type,
		notification.Contact.Value,
		notification.Event.TriggerID,
		notification.Event.Metric,
		notification.Event.State,
		notification.Event.Timestamp,
		notification.Event.GetMetricsValues(),
		notification.SendFail,
		notification.Throttled,
		notification.Timestamp,
	)
}

// IsScheduleAllows check if the time is in the allowed schedule interval
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
	return fmt.Sprintf("TriggerId: %s, Metric: %s, Values: %s, OldState: %s, State: %s, Message: '%s', Timestamp: %v", event.TriggerID, event.Metric, event.GetMetricsValues(), event.OldState, event.State, event.CreateMessage(nil), event.Timestamp)
}

// GetMetricsValues gets event metric value and format it to human readable presentation
func (event NotificationEvent) GetMetricsValues() string {
	var targetNames []string //nolint
	for targetName := range event.Values {
		targetNames = append(targetNames, targetName)
	}
	if len(targetNames) == 0 {
		return "—"
	}
	if len(targetNames) == 1 {
		return strconv.FormatFloat(event.Values[targetNames[0]], 'f', -1, 64)
	}
	var builder strings.Builder
	sort.Strings(targetNames)
	for i, targetName := range targetNames {
		builder.WriteString(targetName)
		builder.WriteString(": ")
		value := strconv.FormatFloat(event.Values[targetName], 'f', -1, 64)
		builder.WriteString(value)
		if i < len(targetNames)-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}

// FormatTimestamp gets event timestamp and format it using given location to human readable presentation
func (event NotificationEvent) FormatTimestamp(location *time.Location) string {
	return time.Unix(event.Timestamp, 0).In(location).Format("15:04")
}

// GetOrCreateMetricState gets metric state from check data or create new if CheckData has no state for given metric
func (checkData *CheckData) GetOrCreateMetricState(metric string, emptyTimestampValue int64, muteNewMetric bool) MetricState {
	_, ok := checkData.Metrics[metric]
	if !ok {
		checkData.Metrics[metric] = createEmptyMetricState(emptyTimestampValue, !muteNewMetric)
	}
	return checkData.Metrics[metric]
}

// SetMaintenance set maintenance user, time for CheckData
func (checkData *CheckData) SetMaintenance(maintenanceInfo *MaintenanceInfo, maintenance int64) {
	checkData.MaintenanceInfo = *maintenanceInfo
	checkData.Maintenance = maintenance
}

// GetMaintenance return metricState MaintenanceInfo
func (checkData *CheckData) GetMaintenance() (MaintenanceInfo, int64) {
	return checkData.MaintenanceInfo, checkData.Maintenance
}

func createEmptyMetricState(defaultTimestampValue int64, firstStateIsNodata bool) MetricState {
	if firstStateIsNodata {
		return MetricState{
			State:     StateNODATA,
			Timestamp: defaultTimestampValue,
		}
	}

	unixNow := time.Now().Unix()

	return MetricState{
		State:          StateOK,
		Timestamp:      unixNow,
		EventTimestamp: unixNow,
	}
}

// GetCheckPoint gets check point for given MetricState
// CheckPoint is the timestamp from which to start checking the current state of the metric
func (metricState *MetricState) GetCheckPoint(checkPointGap int64) int64 {
	return int64(math.Max(float64(metricState.Timestamp-checkPointGap), float64(metricState.EventTimestamp)))
}

// GetEventTimestamp gets event timestamp for given metric
func (metricState MetricState) GetEventTimestamp() int64 {
	if metricState.EventTimestamp == 0 {
		return metricState.Timestamp
	}
	return metricState.EventTimestamp
}

// GetEventTimestamp gets event timestamp for given check
func (checkData CheckData) GetEventTimestamp() int64 {
	if checkData.EventTimestamp == 0 {
		return checkData.Timestamp
	}
	return checkData.EventTimestamp
}

// IsSimple checks triggers patterns
// If patterns more than one or it contains standard graphite wildcard symbols,
// when this target can contain more then one metrics, and is it not simple trigger
func (trigger *Trigger) IsSimple() bool {
	if len(trigger.Targets) > 1 || len(trigger.Patterns) > 1 {
		return false
	}
	for _, pattern := range trigger.Patterns {
		if strings.ContainsAny(pattern, "*{?[") {
			return false
		}
	}
	return true
}

// UpdateScore update and return checkData score, based on metric states and checkData state
func (checkData *CheckData) UpdateScore() int64 {
	checkData.Score = stateScores[checkData.State]
	for _, metricData := range checkData.Metrics {
		checkData.Score += stateScores[metricData.State]
	}
	return checkData.Score
}

// MustIgnore returns true if given state transition must be ignored
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

// isAnonymous checks if user is Anonymous or empty
func isAnonymous(user string) bool {
	return user == "anonymous" || user == ""
}

// SetMaintenanceUserAndTime set startuser and starttime or stopuser and stoptime for MaintenanceInfo
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
