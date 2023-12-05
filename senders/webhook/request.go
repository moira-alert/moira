package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/templating"
)

func (client *webhookClient) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) (*http.Request, error) {
	if client.url == moira.VariableContactValue {
		client.logger.Warning().
			String("potentially_dangerous_url", client.url).
			Msg("Found potentially dangerous url template, api contact validation is advised")
	}

	requestURL := buildRequestURL(client.url, trigger, contact)

	requestBody, err := buildRequestBody(client.body, events, contact, trigger, plots, throttled)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}

	if client.user != "" && client.password != "" {
		request.SetBasicAuth(client.user, client.password)
	}

	for k, v := range client.headers {
		request.Header.Set(k, v)
	}

	client.logger.Debug().
		String("method", request.Method).
		String("url", request.URL.String()).
		String("body", bytes.NewBuffer(requestBody).String()).
		Msg("Created request")

	return request, nil
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

func buildRequestBody(
	template string,
	events moira.NotificationEvents,
	contact moira.ContactData,
	trigger moira.TriggerData,
	plots [][]byte,
	throttled bool,
) ([]byte, error) {
	var requestBody []byte

	if template != "" {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("failed to generate new id: %w", err)
		}

		populatedBody, err := templating.Populate(templating.TemplateSettings{
			ID:      id.String(),
			Desc:    template,
			Name:    trigger.Name,
			Events:  moira.NotificationEventsToTemplatingEvents(events),
			Contact: contact.ToTemplatingContactInfo(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to populate template body: %w", err)
		}

		requestBody = []byte(populatedBody)
	} else {
		defaultBody, err := buildDefaultRequestBody(events, contact, trigger, plots, throttled)
		if err != nil {
			return nil, fmt.Errorf("failed to build default request body: %w", err)
		}

		requestBody = defaultBody
	}

	return requestBody, nil
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
