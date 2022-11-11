package mattermost

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"

	"github.com/mattermost/mattermost-server/v6/model"
)

// Sender posts messages to Mattermost chat.
// It implements moira.Sender.
// You must call Init method before SendEvents method.
type Sender struct {
	frontURI string
	location *time.Location
	client   Client
}

// Init configures Sender.
func (sender *Sender) Init(senderSettings map[string]string, _ moira.Logger, location *time.Location, _ string) error {
	url := senderSettings["url"]
	if url == "" {
		return fmt.Errorf("can not read Mattermost url from config")
	}
	client := model.NewAPIv4Client(url)

	insecureTLS, err := strconv.ParseBool(senderSettings["insecure_tls"])
	if err != nil {
		return fmt.Errorf("can not parse insecure_tls: %v", err)
	}
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureTLS,
			},
		},
	}
	sender.client = client

	token := senderSettings["api_token"]
	if token == "" {
		return fmt.Errorf("can not read Mattermost api_token from config")
	}
	sender.client.SetToken(token)

	frontURI := senderSettings["front_uri"]
	if frontURI == "" {
		return fmt.Errorf("can not read Mattermost front_uri from config")
	}
	sender.frontURI = frontURI
	sender.location = location

	return nil
}

// SendEvents implements moira.Sender interface.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, _ [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	err := sender.sendMessage(message, contact.Value, trigger.ID)
	if err != nil {
		return err
	}

	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	const messageMaxCharacters = 4_000

	var message strings.Builder

	title := sender.buildTitle(events, trigger)
	titleLen := len([]rune(title))

	desc := sender.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := sender.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = sender.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	}
	return desc
}

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("**%s**", events.GetSubjectState())
	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" [%s](%s)", trigger.Name, triggerURI)
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}

	title += "\n"
	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if n is negative buildEventsString does not limit the events string
func (sender *Sender) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString = "```"
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State)
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune("```")) + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "```"

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func (sender *Sender) sendMessage(message string, contact string, triggerID string) error {
	post := model.Post{
		ChannelId: contact,
		Message:   message,
	}

	_, _, err := sender.client.CreatePost(&post)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to Mattermost [%s]: %s", triggerID, contact, err)
	}

	return nil
}
