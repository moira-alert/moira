package webhook

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/moira-alert/moira"
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

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) (*http.Request, error) {
	requestURL := buildRequestURL(sender.url, trigger, contact)
	requestBody, err := buildRequestBody(events, contact, trigger, plot, throttled)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}
	if sender.user != "" && sender.password != "" {
		request.SetBasicAuth(sender.user, sender.password)
	}
	for k, v := range sender.headers {
		request.Header.Set(k, v)
	}
	sender.log.Debugf("%s %s '%s'", request.Method, request.URL.String(), bytes.NewBuffer(requestBody).String())
	return request, nil
}

func buildRequestBody(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) ([]byte, error) {
	eventsData := make([]eventData, 0, len(events))
	for _, event := range events {
		eventsData = append(eventsData, eventData{
			Metric:         event.Metric,
			Value:          moira.UseFloat64(event.Value),
			Timestamp:      event.Timestamp,
			IsTriggerEvent: event.IsTriggerEvent,
			State:          event.State,
			OldState:       event.OldState,
		})
	}
	requestPayload := payload{
		Trigger: triggerData{
			ID:          trigger.ID,
			Name:        trigger.Name,
			Description: trigger.Desc,
			Tags:        trigger.Tags,
		},
		Events: eventsData,
		Contact: contactData{
			Type:  contact.Type,
			Value: contact.Value,
			ID:    contact.ID,
			User:  contact.User,
		},
		Plot:      bytesToBase64(plot),
		Throttled: throttled,
	}
	return json.Marshal(requestPayload)
}

func buildRequestURL(pattern string, trigger moira.TriggerData, contact moira.ContactData) string {
	templateVariables := map[string]string{
		"${contact_id}":    contact.ID,
		"${contact_value}": contact.Value,
		"${contact_type}":  contact.Type,
		"${trigger_id}":    trigger.ID,
	}
	for k, v := range templateVariables {
		pattern = strings.Replace(pattern, k, url.PathEscape(v), -1)
	}
	return pattern
}

// bytesToBase64 converts given bytes slice to base64 string
func bytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
