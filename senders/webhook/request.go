package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/moira-alert/moira"
)

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) (*http.Request, error) {
	webhookClient, ok := sender.webhookClients[contact.Type]
	if !ok {
		return nil, fmt.Errorf("failed to build request because there is not %s client", contact.Type)
	}

	if webhookClient.url == moira.VariableContactValue {
		sender.log.Warning().
			String("potentially_dangerous_url", webhookClient.url).
			Msg("Found potentially dangerous url template, api contact validation is advised")
	}

	requestURL := buildRequestURL(webhookClient.url, trigger, contact)
	requestBody, err := buildRequestBody(events, contact, trigger, plots, throttled)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}

	if webhookClient.user != "" && webhookClient.password != "" {
		request.SetBasicAuth(webhookClient.user, webhookClient.password)
	}

	for k, v := range webhookClient.headers {
		request.Header.Set(k, v)
	}

	sender.log.Debug().
		String("method", request.Method).
		String("url", request.URL.String()).
		String("body", bytes.NewBuffer(requestBody).String()).
		Msg("Created request")

	return request, nil
}

func buildRequestBody(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) ([]byte, error) {
	encodedFirstPlot := ""
	encodedPlots := make([]string, 0, len(plots))
	for i, plot := range plots {
		encodedPlot := bytesToBase64(plot)
		encodedPlots = append(encodedPlots, encodedPlot)
		if i == 0 {
			encodedFirstPlot = encodedPlot
		}
	}
	requestPayload := payload{
		Trigger: toTriggerData(trigger),
		Events:  toEventsData(events),
		Contact: contactData{
			Type:  contact.Type,
			Value: contact.Value,
			ID:    contact.ID,
			User:  contact.User,
			Team:  contact.Team,
		},
		Plot:      encodedFirstPlot,
		Plots:     encodedPlots,
		Throttled: throttled,
	}
	return json.Marshal(requestPayload)
}

func buildRequestURL(template string, trigger moira.TriggerData, contact moira.ContactData) string {
	templateVariables := map[string]string{
		moira.VariableContactID:    contact.ID,
		moira.VariableContactValue: contact.Value,
		moira.VariableContactType:  contact.Type,
		moira.VariableTriggerID:    trigger.ID,
	}
	for k, v := range templateVariables {
		value := url.PathEscape(v)
		if k == moira.VariableContactValue &&
			(strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")) {
			value = v
		}
		template = strings.Replace(template, k, value, -1)
	}
	return template
}
