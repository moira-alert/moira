package msteams

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/moira-alert/moira"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Sender implements moira sender interface via MS Teams
type Sender struct {
	frontURI string
	logger   moira.Logger
	location *time.Location
	client   *http.Client
}

func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	sender.location = location
	sender.frontURI = senderSettings["front_uri"]
	sender.client = &http.Client{
		Timeout:   time.Duration(30) * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {

	valid, err := isValidWebhookURL(contact.Value)
	if valid != true {
		return err
	}

	if err != nil {
		sender.logger.Error(err)
	}

	request, err := sender.buildRequest(events, contact, trigger, plot, throttled)
	if request != nil {
		defer request.Body.Close()
	}

	response, err := sender.client.Do(request)
	if response != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err.Error())
	}

	return nil
}

func isValidWebhookURL(webhookURL string) (bool, error) {
	// basic URL check
	_, err := url.Parse(webhookURL)
	if err != nil {
		return false, err
	}
	// only pass MS teams webhook URLs
	hasPrefix := strings.HasPrefix(webhookURL, "https://outlook.office.com/webhook/")
	if hasPrefix != true {
		err = errors.New("unvalid ms teams webhook url")
		return false, err
	}
	return true, nil
}

func buildMessageCard(title string, uri string, state moira.State, data moira.TriggerData, facts []Fact) MessageCard {
	messageCard := MessageCard{
		Context:     "http://schema.org/extensions",
		MessageType: "MessageCard",
		Summary:     "Moira Alert",
		ThemeColor:  getColourForState(state),
		Title:       title,
		Sections: []Section{
			{
				ActivityTitle: "Description",
				ActivityText:  strings.ReplaceAll(data.Desc, "\n", "  \n\n"),
				Facts:         facts,
			},
		},
		PotentialAction: []Actions{
			{
				Type: "OpenUri",
				Name: "View in Moira",
				Targets: []OpenUriTarget{
					{
						Os:  "default",
						Uri: uri,
					},
				},
			},
		},
	}
	return messageCard
}

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) (*http.Request, error) {

	title, uri := sender.buildTitleAndUri(events, trigger)
	facts := sender.buildEventsFacts(events, -1, throttled)

	messageCard := buildMessageCard(title, uri, events.GetSubjectState(), trigger, facts)
	requestURL := contact.Value
	requestBody, err := json.Marshal(messageCard)
	if err != nil {
		return nil, err
	}

	sender.logger.Infof("%s\n", requestBody)
	request, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}
	headers := map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	sender.logger.Debugf("%s %s '%s'", request.Method, request.URL.String(), string(requestBody))
	return request, nil
}

func (sender *Sender) buildTitleAndUri(events moira.NotificationEvents, trigger moira.TriggerData) (string, string) {
	title := fmt.Sprintf("%s", events.GetSubjectState())

	if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}
	triggerURI := trigger.GetTriggerURI(sender.frontURI)

	return title, triggerURI
}

// buildEventsFacts builds Facts from moira events
// if n is negative buildEventsFacts does not limit the Facts array
func (sender *Sender) buildEventsFacts(events moira.NotificationEvents, maxEvents int, throttled bool) []Fact {
	var facts []Fact

	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("%s = %s (%s to %s)", event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		facts = append(facts, Fact{
			Name:  event.FormatTimestamp(sender.location),
			Value: "```" + line + "```",
			state: event.State,
		})

		if maxEvents != -1 && len(facts) > maxEvents {
			facts = append(facts, Fact{
				Name:  "Info",
				Value: "```" + fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted) + "```",
			})
			break
		}
		eventsPrinted++
	}

	if throttled {
		facts = append(facts, Fact{
			Name:  "Warning",
			Value: "Please, *fix your system or tune this trigger* to generate less events.",
		})
	}
	return facts
}

func getColourForState(state moira.State) string {
	switch state := state; state {
	case moira.StateOK:
		return "008000" //green
	case moira.StateWARN:
		return "ffa500" //orange
	case moira.StateERROR:
		return "ffa500" //red
	case moira.StateNODATA:
		return "000000" //black
	default:
		return "ffffff" //unhandled state
	}
}
