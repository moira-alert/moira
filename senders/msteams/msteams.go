package msteams

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/russross/blackfriday/v2"
)

// Sender implements moira sender interface via MS Teams
type Sender struct {
	frontURI string
	logger   moira.Logger
	location *time.Location
	client   *http.Client
}

// Init initialises settings required for full functionality
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	sender.location = location
	sender.frontURI = senderSettings["front_uri"]
	sender.client = &http.Client{
		Timeout: time.Duration(30) * time.Second,
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {

	err := sender.isValidWebhookURL(contact.Value)
	if err != nil {
		return fmt.Errorf("invalid msteams webhook url: %s", err.Error())
	}

	request, err := sender.buildRequest(events, contact, trigger, plot, throttled)

	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}

	response, err := sender.client.Do(request)

	if err != nil {
		return fmt.Errorf("failed to perform request: %s", err.Error())
	}
	defer response.Body.Close()

	// read the entire response as required by https://golang.org/pkg/net/http/#Client.Do
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to decode response: %s", err.Error())
	}

	//handle non 2xx responses
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode <= http.StatusNetworkAuthenticationRequired {
		return fmt.Errorf("server responded with a non 2xx code: %d", response.StatusCode)
	}

	responseData := string(body)
	if responseData != "1" {
		return fmt.Errorf("teams endpoint responded with an error: %s", errors.New(responseData))
	}

	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) MessageCard {

	title, uri := sender.buildTitleAndURI(events, trigger)
	var triggerDescription string
	if trigger.Desc != "" {
		triggerDescription = string(blackfriday.Run([]byte(trigger.Desc)))
	}
	facts := sender.buildEventsFacts(events, -1, throttled)
	var actions []Actions
	if uri != "" {
		actions = append(actions, Actions{
			Type: "OpenUri",
			Name: "View in Moira",
			Targets: []OpenURITarget{
				{
					Os:  "default",
					URI: uri,
				},
			},
		})
	}
	return MessageCard{
		Context:     "http://schema.org/extensions",
		MessageType: "MessageCard",
		Summary:     "Moira Alert",
		ThemeColor:  getColourForState(events.GetSubjectState()),
		Title:       title,
		Sections: []Section{
			{
				ActivityTitle: "Description",
				ActivityText:  triggerDescription,
				Facts:         facts,
			},
		},
		PotentialAction: actions,
	}
}

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) (*http.Request, error) {

	messageCard := sender.buildMessage(events, trigger, throttled)
	requestURL := contact.Value
	requestBody, err := json.Marshal(messageCard)
	if err != nil {
		return nil, err
	}

	sender.logger.Debugf("%s\n", requestBody)
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

func (sender *Sender) buildTitleAndURI(events moira.NotificationEvents, trigger moira.TriggerData) (string, string) {
	title := string(events.GetSubjectState())

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

func (sender *Sender) isValidWebhookURL(webhookURL string) error {
	// basic URL check
	_, err := url.Parse(webhookURL)
	if err != nil {
		return err
	}
	// only pass MS teams webhook URLs
	hasPrefix := strings.HasPrefix(webhookURL, "https://outlook.office.com/webhook/")
	if !hasPrefix {
		err = errors.New("unvalid ms teams webhook url")
		return err
	}
	return nil
}

func getColourForState(state moira.State) string {
	switch state := state; state {
	case moira.StateOK:
		return "008000" //green
	case moira.StateWARN:
		return "ffa500" //orange
	case moira.StateERROR:
		return "ff0000" //red
	case moira.StateNODATA:
		return "000000" //black
	default:
		return "ffffff" //unhandled state
	}
}
