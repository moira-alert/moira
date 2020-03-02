package webhook

import (
	"encoding/base64"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

type payload struct {
	Trigger   triggerData `json:"trigger"`
	Events    []eventData `json:"events"`
	Contact   contactData `json:"contact"`
	Plot      string      `json:"plot"`
	Throttled bool        `json:"throttled"`
}

type triggerData struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type eventData struct {
	Metric         string  `json:"metric"`
	Value          float64 `json:"value"`
	Timestamp      int64   `json:"timestamp"`
	IsTriggerEvent bool    `json:"trigger_event"`
	State          string  `json:"state"`
	OldState       string  `json:"old_state"`
}

type contactData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id"`
	User  string `json:"user"`
}

// toTriggerData returns correct triggerData structure to marshall JSON
func toTriggerData(trigger moira2.TriggerData) triggerData {
	result := triggerData{
		ID:          trigger.ID,
		Name:        trigger.Name,
		Description: trigger.Desc,
		Tags:        make([]string, 0),
	}
	result.Tags = append(result.Tags, trigger.Tags...)
	return result
}

// toEventsData returns correct eventData structure collection to marshall JSON
func toEventsData(events moira2.NotificationEvents) []eventData {
	result := make([]eventData, 0, len(events))
	for _, event := range events {
		result = append(result, eventData{
			Metric:         event.Metric,
			Value:          moira2.UseFloat64(event.Value),
			Timestamp:      event.Timestamp,
			IsTriggerEvent: event.IsTriggerEvent,
			State:          event.State.String(),
			OldState:       event.OldState.String(),
		})
	}
	return result
}

// bytesToBase64 converts given bytes slice to base64 string
func bytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
