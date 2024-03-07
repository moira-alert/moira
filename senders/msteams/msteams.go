package msteams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/russross/blackfriday/v2"
)

const extensions = "http://schema.org/extensions"
const messageType = "MessageCard"
const summary = "Moira Alert"
const teamsBaseURL = "https://outlook.office.com/webhook/"
const teamsOKResponse = "1"
const openURI = "OpenUri"
const openURIMessage = "View in Moira"
const openURIOsDefault = "default"
const activityTitleText = "Description"

var throttleWarningFact = Fact{
	Name:  "Warning",
	Value: "Please, *fix your system or tune this trigger* to generate less events.",
}

var headers = map[string]string{
	"User-Agent":   "Moira",
	"Content-Type": "application/json",
}

// Structure that represents the MSTeams configuration in the YAML file.
type config struct {
	FrontURI  string `mapstructure:"front_uri"`
	MaxEvents int    `mapstructure:"max_events"`
}

// Sender implements moira sender interface via MS Teams.
type Sender struct {
	frontURI  string
	maxEvents int
	logger    moira.Logger
	location  *time.Location
	client    *http.Client
}

// Init initialises settings required for full functionality.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to msteams config: %w", err)
	}

	sender.logger = logger
	sender.location = location
	sender.frontURI = cfg.FrontURI
	sender.maxEvents = cfg.MaxEvents
	sender.client = &http.Client{
		Timeout: time.Duration(30) * time.Second, //nolint
	}
	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	err := sender.isValidWebhookURL(contact.Value)
	if err != nil {
		return err
	}

	request, err := sender.buildRequest(events, contact, trigger, throttled)

	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	response, err := sender.client.Do(request)

	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}
	defer response.Body.Close()

	// read the entire response as required by https://golang.org/pkg/net/http/#Client.Do
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// handle non 2xx responses
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode <= http.StatusNetworkAuthenticationRequired {
		return fmt.Errorf("server responded with a non 2xx code: %d", response.StatusCode)
	}

	responseData := string(body)
	if responseData != teamsOKResponse {
		return fmt.Errorf("teams endpoint responded with an error: %s", responseData)
	}

	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) MessageCard {
	title, uri := sender.buildTitleAndURI(events, trigger, throttled)
	var triggerDescription string
	if trigger.Desc != "" {
		triggerDescription = string(blackfriday.Run([]byte(trigger.Desc)))
	}
	facts := sender.buildEventsFacts(events, sender.maxEvents, throttled)
	var actions []Action
	if uri != "" {
		actions = append(actions, Action{
			Type: openURI,
			Name: openURIMessage,
			Targets: []OpenURITarget{
				{
					Os:  openURIOsDefault,
					URI: uri,
				},
			},
		})
	}

	state := events.GetCurrentState(throttled)

	return MessageCard{
		Context:     extensions,
		MessageType: messageType,
		Summary:     summary,
		ThemeColor:  getColourForState(state),
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

func (sender *Sender) buildRequest(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) (*http.Request, error) {
	messageCard := sender.buildMessage(events, trigger, throttled)
	requestURL := contact.Value
	requestBody, err := json.Marshal(messageCard)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return request, err
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}
	sender.logger.Debug().
		String("payload", string(requestBody)).
		String("endpoint", request.URL.String()).
		Msg("Created payload for teams endpoint")

	return request, nil
}

func (sender *Sender) buildTitleAndURI(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) (string, string) {
	state := events.GetCurrentState(throttled)

	title := string(state)

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
// if n is negative buildEventsFacts does not limit the Facts array.
func (sender *Sender) buildEventsFacts(events moira.NotificationEvents, maxEvents int, throttled bool) []Fact {
	var facts []Fact //nolint

	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("%s = %s (%s to %s)", event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		facts = append(facts, Fact{
			Name:  event.FormatTimestamp(sender.location, moira.DefaultTimeFormat),
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

func getColourForState(state moira.State) string {
	switch state := state; state {
	case moira.StateOK:
		return Green
	case moira.StateWARN:
		return Orange
	case moira.StateERROR:
		return Red
	case moira.StateNODATA:
		return Black
	default:
		return White // unhandled state
	}
}
