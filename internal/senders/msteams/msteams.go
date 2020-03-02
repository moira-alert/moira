package msteams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/russross/blackfriday/v2"
)

const context = "http://schema.org/extensions"
const messageType = "MessageCard"
const summary = "Moira Alert"
const teamsBaseURL = "https://outlook.office.com/webhook/"
const teamsOKResponse = "1"
const openUri = "OpenUri"
const openUriMessage = "View in Moira"
const openUriOsDefault = "default"
const activityTitleText = "Description"

var throttleWarningFact = Fact{
	Name:  "Warning",
	Value: "Please, *fix your system or tune this trigger* to generate less events.",
}

var headers = map[string]string{
	"User-Agent":   "Moira",
	"Content-Type": "application/json",
}

// Sender implements moira sender interface via MS Teams
type Sender struct {
	frontURI  string
	maxEvents int
	logger    moira2.Logger
	location  *time.Location
	client    *http.Client
}

// Init initialises settings required for full functionality
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	sender.location = location
	sender.frontURI = senderSettings["front_uri"]
	maxEvents, err := strconv.Atoi(senderSettings["max_events"])
	if err != nil {
		return fmt.Errorf("max_events should be an integer: %w", err)
	}
	sender.maxEvents = maxEvents
	sender.client = &http.Client{
		Timeout: time.Duration(30) * time.Second,
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {

	err := sender.isValidWebhookURL(contact.Value)
	if err != nil {
		return err
	}

	request, err := sender.buildRequest(events, contact, trigger, plot, throttled)

	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	response, err := sender.client.Do(request)

	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}
	defer response.Body.Close()

	// read the entire response as required by https://golang.org/pkg/net/http/#Client.Do
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	//handle non 2xx responses
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode <= http.StatusNetworkAuthenticationRequired {
		return fmt.Errorf("server responded with a non 2xx code: %d", response.StatusCode)

	}

	responseData := string(body)
	if responseData != teamsOKResponse {
		return fmt.Errorf("teams endpoint responded with an error: %s", responseData)
	}

	return nil
}

func (sender *Sender) buildMessage(events moira2.NotificationEvents, trigger moira2.TriggerData, throttled bool) MessageCard {

	title, uri := sender.buildTitleAndURI(events, trigger)
	var triggerDescription string
	if trigger.Desc != "" {
		triggerDescription = string(blackfriday.Run([]byte(trigger.Desc)))
	}
	facts := sender.buildEventsFacts(events, sender.maxEvents, throttled)
	var actions []Action
	if uri != "" {
		actions = append(actions, Action{
			Type: openUri,
			Name: openUriMessage,
			Targets: []OpenURITarget{
				{
					Os:  openUriOsDefault,
					URI: uri,
				},
			},
		})
	}

	return MessageCard{
		Context:     context,
		MessageType: messageType,
		Summary:     summary,
		ThemeColor:  getColourForState(events.GetSubjectState()),
		Title:       title,
		Sections: []Section{
			{
				ActivityTitle: activityTitleText,
				ActivityText:  triggerDescription,
				Facts:         facts,
			},
		},
		PotentialAction: actions,
	}
}

func (sender *Sender) buildRequest(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) (*http.Request, error) {

	messageCard := sender.buildMessage(events, trigger, throttled)
	requestURL := contact.Value
	requestBody, err := json.Marshal(messageCard)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}
	sender.logger.Debugf("created payload '%s' for teams endpoint %s", string(requestBody), request.URL.String())
	return request, nil
}

func (sender *Sender) buildTitleAndURI(events moira2.NotificationEvents, trigger moira2.TriggerData) (string, string) {
	title := string(events.GetSubjectState())

	if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title = fmt.Sprintf("%s %s", title, tags)
	}
	triggerURI := trigger.GetTriggerURI(sender.frontURI)

	return title, triggerURI
}

// buildEventsFacts builds Facts from moira events
// if n is negative buildEventsFacts does not limit the Facts array
func (sender *Sender) buildEventsFacts(events moira2.NotificationEvents, maxEvents int, throttled bool) []Fact {
	var facts []Fact

	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("%s = %s (%s to %s)", event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira2.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira2.UseString(event.Message))
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
		facts = append(facts, throttleWarningFact)
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
	hasPrefix := strings.HasPrefix(webhookURL, teamsBaseURL)
	if !hasPrefix {
		return fmt.Errorf("%s is an invalid ms teams webhook url", webhookURL)
	}
	return nil
}

func getColourForState(state moira2.State) string {
	switch state := state; state {
	case moira2.StateOK:
		return Green
	case moira2.StateWARN:
		return Orange
	case moira2.StateERROR:
		return Red
	case moira2.StateNODATA:
		return Black
	default:
		return White //unhandled state
	}
}
