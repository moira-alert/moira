package moira_alert

import (
	"bytes"
	"fmt"
)

var (
	eventStates      = [...]string{"OK", "WARN", "ERROR", "NODATA", "TEST"}
	eventStateWeight = map[string]int{
		"OK":     0,
		"WARN":   1,
		"ERROR":  100,
		"NODATA": 10000,
	}
	eventHighDegradationTag = "HIGH DEGRADATION"
	eventDegradationTag     = "DEGRADATION"
	eventProgressTag        = "PROGRESS"
)

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
	Enabled           bool         `json:"enabled"`
	Tags              []string     `json:"tags"`
	Schedule          ScheduleData `json:"sched"`
	ID                string       `json:"id"`
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

// GetSubjectState returns the most critial state of events
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

//GetPseudoTags returns additional subscription tags based on trigger state
func (event *EventData) GetPseudoTags() []string {
	tags := []string{event.State, event.OldState}
	if oldStateWeight, ok := eventStateWeight[event.OldState]; ok {
		if newStateWeight, ok := eventStateWeight[event.State]; ok {
			if newStateWeight > oldStateWeight {
				if newStateWeight-oldStateWeight >= 100 {
					tags = append(tags, eventHighDegradationTag)
				}
				tags = append(tags, eventDegradationTag)
			}
			if newStateWeight < oldStateWeight {
				tags = append(tags, eventProgressTag)
			}
		}
	}
	return tags
}
