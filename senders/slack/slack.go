package slack

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	slackdown "github.com/karriereat/blackfriday-slack"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	blackfriday "github.com/russross/blackfriday/v2"

	slack_client "github.com/slack-go/slack"
)

const (
	okEmoji        = ":moira-state-ok:"
	warnEmoji      = ":moira-state-warn:"
	errorEmoji     = ":moira-state-error:"
	nodataEmoji    = ":moira-state-nodata:"
	exceptionEmoji = ":moira-state-exception:"
	testEmoji      = ":moira-state-test:"

	messageMaxCharacters = 4000

	//see errors https://api.slack.com/methods/chat.postMessage
	ErrorTextChannelArchived = "is_archived"
	ErrorTextChannelNotFound = "channel_not_found"
	ErrorTextNotInChannel    = "not_in_channel"
)

var stateEmoji = map[moira.State]string{
	moira.StateOK:        okEmoji,
	moira.StateWARN:      warnEmoji,
	moira.StateERROR:     errorEmoji,
	moira.StateNODATA:    nodataEmoji,
	moira.StateEXCEPTION: exceptionEmoji,
	moira.StateTEST:      testEmoji,
}

// Structure that represents the Slack configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	APIToken string `mapstructure:"api_token"`
	UseEmoji bool   `mapstructure:"use_emoji"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface via slack
type Sender struct {
	clients map[string]*slackClient
}

type slackClient struct {
	frontURI string
	useEmoji bool
	logger   moira.Logger
	location *time.Location
	client   *slack_client.Client
}

// Init read yaml config
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to slack config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read slack api_token from config")
	}

	client := &slackClient{
		useEmoji: cfg.UseEmoji,
		logger:   opts.Logger,
		frontURI: cfg.FrontURI,
		location: opts.Location,
		client:   slack_client.New(cfg.APIToken),
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*slackClient)
	}

	sender.clients[senderIdent] = client

	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	slackClient, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	message := slackClient.buildMessage(events, trigger, throttled)
	useDirectMessaging := useDirectMessaging(contact.Value)

	state := events.GetCurrentState(throttled)
	emoji := slackClient.getStateEmoji(state)

	channelID, threadTimestamp, err := slackClient.sendMessage(message, contact.Value, trigger.ID, useDirectMessaging, emoji)
	if err != nil {
		return err
	}

	if channelID != "" && len(plots) > 0 {
		err = slackClient.sendPlots(plots, channelID, threadTimestamp, trigger.ID)
		if err != nil {
			slackClient.logger.Warning().
				String("trigger_id", trigger.ID).
				String("contact_value", contact.Value).
				String("contact_type", contact.Type).
				Error(err)
		}
	}

	return nil
}

func (client *slackClient) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder

	title := client.buildTitle(events, trigger, throttled)
	titleLen := len([]rune(title))

	desc := client.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := client.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = client.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (client *slackClient) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = string(blackfriday.Run([]byte(desc), blackfriday.WithRenderer(&slackdown.Renderer{})))
		desc += "\n"
	}
	return desc
}

func (client *slackClient) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := fmt.Sprintf("*%s*", state)
	triggerURI := trigger.GetTriggerURI(client.frontURI)

	if triggerURI != "" {
		title += fmt.Sprintf(" <%s|%s>", triggerURI, trigger.Name)
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
func (client *slackClient) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += "```"
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(client.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
		if msg := event.CreateMessage(client.location); len(msg) > 0 {
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

func (client *slackClient) sendMessage(message string, contact string, triggerID string, useDirectMessaging bool, emoji string) (string, string, error) {
	params := slack_client.PostMessageParameters{
		Username:  "Moira",
		AsUser:    useDirectMessaging,
		IconEmoji: emoji,
		Markdown:  true,
		LinkNames: 1,
	}
	client.logger.Debug().
		String("message", message).
		Msg("Calling slack")

	channelID, threadTimestamp, err := client.client.PostMessage(contact, slack_client.MsgOptionText(message, false), slack_client.MsgOptionPostMessageParameters(params))
	if err != nil {
		errorText := err.Error()
		if errorText == ErrorTextChannelArchived || errorText == ErrorTextNotInChannel ||
			errorText == ErrorTextChannelNotFound {
			return channelID, threadTimestamp, moira.NewSenderBrokenContactError(err)
		}

		return channelID, threadTimestamp, fmt.Errorf("failed to send %s event message to slack [%s]: %s",
			triggerID, contact, errorText)
	}

	return channelID, threadTimestamp, nil
}

func (client *slackClient) sendPlots(plots [][]byte, channelID, threadTimestamp, triggerID string) error {
	filename := fmt.Sprintf("%s.png", triggerID)
	for _, plot := range plots {
		reader := bytes.NewReader(plot)
		uploadParameters := slack_client.UploadFileV2Parameters{
			FileSize:        len(plot),
			Reader:          reader,
			Title:           filename,
			Filename:        filename,
			Channel:         channelID,
			ThreadTimestamp: threadTimestamp,
		}

		_, err := client.client.UploadFileV2(uploadParameters)
		if err != nil {
			return err
		}
	}

	return nil
}

// getStateEmoji returns corresponding state emoji
func (client *slackClient) getStateEmoji(subjectState moira.State) string {
	if client.useEmoji {
		if emoji, ok := stateEmoji[subjectState]; ok {
			return emoji
		}
	}

	return slack_client.DEFAULT_MESSAGE_ICON_EMOJI
}

// useDirectMessaging returns true if user contact is provided
func useDirectMessaging(contactValue string) bool {
	return len(contactValue) > 0 && contactValue[0:1] == "@"
}
