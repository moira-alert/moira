package moira

import (
	"bytes"
	"fmt"
)

var (
	eventStates = [...]string{"OK", "WARN", "ERROR", "NODATA", "TEST"}
)

// EventData represents trigger state changes event
type EventData struct {
	Timestamp      int64   `json:"timestamp"`
	Metric         string  `json:"metric"`
	Value          float64 `json:"value"`
	State          string  `json:"state"`
	TriggerID      string  `json:"trigger_id"`
	SubscriptionID string  `json:"sub_id"`
	OldState       string  `json:"old_state"`
	Message        string  `json:"msg"`
}

// EventsData represents slice of EventData
type EventsData []EventData

// TriggerData represents trigger object
type TriggerData struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Targets    []string `json:"targets"`
	WarnValue  float64  `json:"warn_value"`
	ErrorValue float64  `json:"error_value"`
	Tags       []string `json:"__notifier_trigger_tags"`
}

// ContactData represents contact object
type ContactData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id"`
	User  string `json:"user"`
}

//SubscriptionData represent user subscription
type SubscriptionData struct {
	Contacts          []string     `json:"contacts"`
	Tags              []string     `json:"tags"`
	Schedule          ScheduleData `json:"sched"`
	ID                string       `json:"id"`
	Enabled           bool         `json:"enabled"`
	ThrottlingEnabled bool         `json:"throttling"`
}

// ScheduleData represent subscription schedule
type ScheduleData struct {
	Days           []ScheduleDataDay `json:"days"`
	TimezoneOffset int64             `json:"tzOffset"`
	StartOffset    int64             `json:"startOffset"`
	EndOffset      int64             `json:"endOffset"`
}

// ScheduleDataDay represent week day of schedule
type ScheduleDataDay struct {
	Enabled bool `json:"enabled"`
}

// ScheduledNotification represent notification object
type ScheduledNotification struct {
	Event     EventData   `json:"event"`
	Trigger   TriggerData `json:"trigger"`
	Contact   ContactData `json:"contact"`
	Throttled bool        `json:"throttled"`
	SendFail  int         `json:"send_fail"`
	Timestamp int64       `json:"timestamp"`
}

// GetSubjectState returns the most critical state of events
func (events EventsData) GetSubjectState() string {
	result := ""
	states := make(map[string]bool)
	for _, event := range events {
		states[event.State] = true
	}
	for _, state := range eventStates {
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
	return fmt.Sprintf("%s:%s:%s:%s:%s:%d:%f:%d:%t:%d",
		notification.Contact.Type,
		notification.Contact.Value,
		notification.Event.TriggerID,
		notification.Event.Metric,
		notification.Event.State,
		notification.Event.Timestamp,
		notification.Event.Value,
		notification.SendFail,
		notification.Throttled,
		notification.Timestamp,
	)
}

// MatchedMetric represent parsed and matched metric data
type MatchedMetric struct {
	Metric             string
	Patterns           []string
	Value              float64
	Timestamp          int64
	RetentionTimestamp int64
	Retention          int
}
