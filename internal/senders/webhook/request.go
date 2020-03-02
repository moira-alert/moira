package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

func (sender *Sender) buildRequest(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) (*http.Request, error) {
	if sender.url == moira2.VariableContactValue {
		sender.log.Warningf("%s is potentially dangerous url template, api contact validation is advised", sender.url)
	}
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

func buildRequestBody(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) ([]byte, error) {
	requestPayload := payload{
		Trigger: toTriggerData(trigger),
		Events:  toEventsData(events),
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

func buildRequestURL(template string, trigger moira2.TriggerData, contact moira2.ContactData) string {
	templateVariables := map[string]string{
		moira2.VariableContactID:    contact.ID,
		moira2.VariableContactValue: contact.Value,
		moira2.VariableContactType:  contact.Type,
		moira2.VariableTriggerID:    trigger.ID,
	}
	for k, v := range templateVariables {
		template = strings.Replace(template, k, url.PathEscape(v), -1)
	}
	return template
}
