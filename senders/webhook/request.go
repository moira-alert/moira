package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/moira-alert/moira"
)

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) (*http.Request, error) {
	if sender.url == moira.VariableContactValue {
		// TODO
		sender.log.Warningf("%s is potentially dangerous url template, api contact validation is advised", sender.url)
	}
	requestURL := buildRequestURL(sender.url, trigger, contact)
	requestBody, err := buildRequestBody(events, contact, trigger, plots, throttled)
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
