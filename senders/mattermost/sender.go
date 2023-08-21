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
	"github.com/mitchellh/mapstructure"
)

// Structure that represents the Mattermost configuration in the YAML file
type mattermost struct {
	Url         string `mapstructure:"url"`
	InsecureTLS string `mapstructure:"insecure_tls"`
	APIToken    string `mapstructure:"api_token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Sender posts messages to Mattermost chat.
// It implements moira.Sender.
// You must call Init method before SendEvents method.
type Sender struct {
	frontURI string
	logger   moira.Logger
	location *time.Location
	client   Client
}

// Init configures Sender.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, _ string) error {
	var mm mattermost
	err := mapstructure.Decode(senderSettings, &mm)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to mattermost config: %w", err)
	}
	url := mm.Url
	if url == "" {
		return fmt.Errorf("can not read Mattermost url from config")
	}
	client := model.NewAPIv4Client(url)

	insecureTLS, err := strconv.ParseBool(mm.InsecureTLS)
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

	token := mm.APIToken
	if token == "" {
		return fmt.Errorf("can not read Mattermost api_token from config")
	}
	sender.client.SetToken(token)

	frontURI := mm.FrontURI
	if frontURI == "" {
		return fmt.Errorf("can not read Mattermost front_uri from config")
	}
	sender.frontURI = frontURI
	sender.location = location
	sender.logger = logger

	return nil
}

// SendEvents implements moira.Sender interface.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	postID, err := sender.sendMessage(message, contact.Value, trigger.ID)
	if err != nil {
		return err
	}
	if postID != "" && len(plots) > 0 {
		err = sender.sendPlots(plots, contact.Value, postID, trigger.ID)
		if err != nil {
			sender.logger.Warning().
				String("trigger_id", trigger.ID).
				String("contact_value", contact.Value).
				String("contact_type", contact.Type).
				Error(err)
		}
	}

	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	const messageMaxCharacters = 4_000

	var message strings.Builder

	title := sender.buildTitle(events, trigger, throttled)
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

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := fmt.Sprintf("**%s**", state)
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
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
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

func (sender *Sender) sendMessage(message string, contact string, triggerID string) (string, error) {
	post := model.Post{
		ChannelId: contact,
		Message:   message,
	}

	sentPost, _, err := sender.client.CreatePost(&post)
	if err != nil {
		return "", fmt.Errorf("failed to send %s event message to Mattermost [%s]: %s", triggerID, contact, err)
	}

	return sentPost.Id, nil
}

func (sender *Sender) sendPlots(plots [][]byte, channelID, postID, triggerID string) error {
	var filesID []string

	filename := fmt.Sprintf("%s.png", triggerID)
	for _, plot := range plots {
		file, _, err := sender.client.UploadFile(plot, channelID, filename)
		if err != nil {
			return err
		}
		for _, info := range file.FileInfos {
			filesID = append(filesID, info.Id)
		}
	}

	if len(filesID) > 0 {
		_, _, err := sender.client.CreatePost(
			&model.Post{
				ChannelId: channelID,
				RootId:    postID,
				FileIds:   filesID,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}
