package webhook

import (
	"encoding/base64"

	"github.com/moira-alert/moira"
)

type payload struct {
	Trigger   triggerData `json:"trigger"`
	Events    []eventData `json:"events"`
	Contact   contactData `json:"contact"`
	Plot      string      `json:"plot"` // Compatibility with Moira < 2.6.0
	Plots     []string    `json:"plots"`
	Throttled bool        `json:"throttled"`
}

type triggerData struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type eventData struct {
	Metric         string             `json:"metric"`
	Values         map[string]float64 `json:"values"`
	Timestamp      int64              `json:"timestamp"`
	IsTriggerEvent bool               `json:"trigger_event"`
	State          moira.State        `json:"state"`
	OldState       moira.State        `json:"old_state"`
}

type contactData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id"`
	User  string `json:"user"`
	Team  string `json:"team"`
}

// toTriggerData returns correct triggerData structure to marshall JSON.
func toTriggerData(trigger moira.TriggerData) triggerData {
	result := triggerData{
		ID:          trigger.ID,
		Name:        trigger.Name,
		Description: trigger.Desc,
		Tags:        make([]string, 0),
	}
	result.Tags = append(result.Tags, trigger.Tags...)

	return result
}

// toEventsData returns correct eventData structure collection to marshall JSON.
func toEventsData(events moira.NotificationEvents) []eventData {
	result := make([]eventData, 0, len(events))
	for _, event := range events {
		result = append(result, eventData{
			Metric:         event.Metric,
			Values:         event.Values,
			Timestamp:      event.Timestamp,
			IsTriggerEvent: event.IsTriggerEvent,
			State:          event.State,
			OldState:       event.OldState,
		})
	}

	return result
}

// bytesToBase64 converts given bytes slice to base64 string.
func bytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
