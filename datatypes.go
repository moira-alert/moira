package moira

import (
	"bytes"
	"fmt"
	"time"
)

var (
	eventStates = [...]string{"OK", "WARN", "ERROR", "NODATA", "TEST"}
)

// NotificationEvent represents trigger state changes event
type NotificationEvent struct {
	Timestamp      int64    `json:"timestamp"`
	Metric         string   `json:"metric"`
	Value          *float64 `json:"value,omitempty"`
	State          string   `json:"state"`
	TriggerID      string   `json:"trigger_id"`
	SubscriptionID *string  `json:"sub_id,omitempty"`
	OldState       string   `json:"old_state"`
	Message        *string  `json:"msg,omitempty"`
}

// EventsData represents slice of NotificationEvent
type EventsData []NotificationEvent

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
	User              string       `json:"user"`
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
	Enabled bool   `json:"enabled"`
	Name    string `json:"name,omitempty"`
}

// ScheduledNotification represent notification object
type ScheduledNotification struct {
	Event     NotificationEvent `json:"event"`
	Trigger   TriggerData       `json:"trigger"`
	Contact   ContactData       `json:"contact"`
	Throttled bool              `json:"throttled"`
	SendFail  int               `json:"send_fail"`
	Timestamp int64             `json:"timestamp"`
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

//MetricValue represent metric data
type MetricValue struct {
	RetentionTimestamp int64   `json:"step,omitempty"`
	Timestamp          int64   `json:"ts"`
	Value              float64 `json:"value"`
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
		UseFloat64(notification.Event.Value),
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
	timestamp := ts - ts%60 - schedule.TimezoneOffset*60
	date := time.Unix(timestamp, 0).UTC()
	if !schedule.Days[int(date.Weekday()+6)%7].Enabled {
		return false
	}
	dayStart := time.Unix(timestamp-timestamp%(24*3600), 0).UTC()
	startDayTime := dayStart.Add(time.Duration(schedule.StartOffset) * time.Minute)
	endDayTime := dayStart.Add(time.Duration(schedule.EndOffset) * time.Minute)
	if schedule.EndOffset < 24*60 {
		if date.After(startDayTime) && date.Before(endDayTime) {
			return true
		}
	} else {
		endDayTime = endDayTime.Add(-time.Hour * 24)
		if date.Before(endDayTime) || date.After(startDayTime) {
			return true
		}
	}
	return false
}

func (eventData NotificationEvent) String() string {
	return fmt.Sprintf("TriggerId: %s, Metric: %s, Value: %v, OldState: %s, State: %s, Message: %s, Timestamp: %v", eventData.TriggerID, eventData.Metric, UseFloat64(eventData.Value), eventData.OldState, eventData.State, UseString(eventData.Message), eventData.Timestamp)
}
