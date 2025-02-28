package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/templating"
)

func buildRequest(
	logger moira.Logger,
	method string,
	requestURL string,
	body []byte,
	user string,
	password string,
	headers map[string]string,
) (*http.Request, error) {
	request, err := http.NewRequest(method, requestURL, bytes.NewBuffer(body))
	if err != nil {
		return request, err
	}

	if user != "" && password != "" {
		request.SetBasicAuth(user, password)
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	logger.Debug().
		String("method", request.Method).
		String("url", request.URL.String()).
		String("body", string(body)).
		Msg("Created request")

	return request, nil
}

func (sender *Sender) buildSendAlertRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) (*http.Request, error) {
	if sender.url == moira.VariableContactValue {
		sender.log.Warning().
			String("potentially_dangerous_url", sender.url).
			Msg("Found potentially dangerous url template, api contact validation is advised")
	}

	requestURL := buildRequestURL(sender.url, trigger, contact)
	requestBody, err := sender.buildRequestBody(events, contact, trigger, plots, throttled)
	if err != nil {
		return nil, err
	}

	return buildRequest(sender.log, http.MethodPost, requestURL, requestBody, sender.user, sender.password, sender.headers)
}

func (sender *Sender) buildRequestBody(
	events moira.NotificationEvents,
	contact moira.ContactData,
	trigger moira.TriggerData,
	plots [][]byte,
	throttled bool,
) ([]byte, error) {
	if sender.body == "" {
		return buildDefaultRequestBody(events, contact, trigger, plots, throttled)
	}

	webhookBodyPopulater := templating.NewWebhookBodyPopulater(contact.ToTemplateContact())
	populatedBody, err := webhookBodyPopulater.Populate(sender.body)
	if err != nil {
		return nil, err
	}

	return []byte(html.UnescapeString(populatedBody)), nil
}

func buildDefaultRequestBody(
	events moira.NotificationEvents,
	contact moira.ContactData,
	trigger moira.TriggerData,
	plots [][]byte,
	throttled bool,
) ([]byte, error) {
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
		template = strings.ReplaceAll(template, k, value)
	}

	return template
}

func (sender *Sender) buildDeliveryCheckRequest(checkData deliveryCheckData) (*http.Request, error) {
	return buildRequest(sender.log, http.MethodGet, checkData.URL, nil, sender.deliveryConfig.User, sender.deliveryConfig.Password, sender.deliveryConfig.Headers)
}

func performRequest(client *http.Client, request *http.Request) (int, []byte, error) {
	rsp, err := client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer rsp.Body.Close()

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return rsp.StatusCode, bodyBytes, nil
}

func (sender *Sender) doCheckRequest(checkData deliveryCheckData) (int, map[string]interface{}, error) {
	req, err := sender.buildDeliveryCheckRequest(checkData)
	if err != nil {
		return 0, nil, err
	}

	statusCode, body, err := performRequest(sender.client, req)
	if err != nil {
		return 0, nil, fmt.Errorf("check delivery request failed: %w", err)
	}

	var rspMap map[string]interface{}
	err = json.Unmarshal(body, &rspMap)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to unmarshal body into json: %w", err)
	}

	return statusCode, rspMap, nil
}
