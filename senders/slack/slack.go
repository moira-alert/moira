package slack

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	slackdown "github.com/karriereat/blackfriday-slack"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	blackfriday "github.com/russross/blackfriday/v2"

	"github.com/nlopes/slack"
)

const (
	okEmoji        = ":moira-state-ok:"
	warnEmoji      = ":moira-state-warn:"
	errorEmoji     = ":moira-state-error:"
	nodataEmoji    = ":moira-state-nodata:"
	exceptionEmoji = ":moira-state-exception:"
	testEmoji      = ":moira-state-test:"

	messageMaxCharacters = 4000
)

var stateEmoji = map[moira.State]string{
	moira.StateOK:        okEmoji,
	moira.StateWARN:      warnEmoji,
	moira.StateERROR:     errorEmoji,
	moira.StateNODATA:    nodataEmoji,
	moira.StateEXCEPTION: exceptionEmoji,
	moira.StateTEST:      testEmoji,
}

// Sender implements moira sender interface via slack
type Sender struct {
	frontURI string
	useEmoji bool
	logger   moira.Logger
	location *time.Location
	client   *slack.Client
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	apiToken := senderSettings["api_token"]
	if apiToken == "" {
		return fmt.Errorf("can not read slack api_token from config")
	}
	sender.useEmoji, _ = strconv.ParseBool(senderSettings["use_emoji"])
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location
	sender.client = slack.New(apiToken)
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	useDirectMessaging := useDirectMessaging(contact.Value)
	emoji := sender.getStateEmoji(events.GetSubjectState())
	channelID, threadTimestamp, err := sender.sendMessage(message, contact.Value, trigger.ID, useDirectMessaging, emoji)
	if err != nil {
		return err
	}
	if channelID != "" && len(plot) > 0 {
		sender.sendPlot(plot, channelID, threadTimestamp, trigger.ID)
	}
	return nil
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message strings.Builder

	title := senders.BuildTitle(events, trigger, sender.frontURI)
	titleLen := len([]rune(title))

	desc := sender.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := senders.BuildEventsString(events, -1, throttled, sender.location)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = senders.BuildEventsString(events, eventsNewLen, throttled, sender.location)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (sender *Sender) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = string(blackfriday.Run([]byte(desc), blackfriday.WithRenderer(&slackdown.Renderer{})))
		desc += "\n"
	}
	return desc
}

func (sender *Sender) sendMessage(message string, contact string, triggerID string, useDirectMessaging bool, emoji string) (string, string, error) {
	params := slack.PostMessageParameters{
		Username:  "Moira",
		AsUser:    useDirectMessaging,
		IconEmoji: emoji,
		Markdown:  true,
	}
	sender.logger.Debugf("Calling slack with message body %s", message)
	channelID, threadTimestamp, err := sender.client.PostMessage(contact, slack.MsgOptionText(message, false), slack.MsgOptionPostMessageParameters(params))
	if err != nil {
		return channelID, threadTimestamp, fmt.Errorf("failed to send %s event message to slack [%s]: %s", triggerID, contact, err.Error())
	}
	return channelID, threadTimestamp, nil
}

func (sender *Sender) sendPlot(plot []byte, channelID, threadTimestamp, triggerID string) error {
	reader := bytes.NewReader(plot)
	uploadParameters := slack.FileUploadParameters{
		Channels:        []string{channelID},
		ThreadTimestamp: threadTimestamp,
		Reader:          reader,
		Filetype:        "png",
		Filename:        fmt.Sprintf("%s.png", triggerID),
	}
	_, err := sender.client.UploadFile(uploadParameters)
	return err
}

// getStateEmoji returns corresponding state emoji
func (sender *Sender) getStateEmoji(subjectState moira.State) string {
	if sender.useEmoji {
		if emoji, ok := stateEmoji[subjectState]; ok {
			return emoji
		}
	}
	return slack.DEFAULT_MESSAGE_ICON_EMOJI
}

// useDirectMessaging returns true if user contact is provided
func useDirectMessaging(contactValue string) bool {
	return len(contactValue) > 0 && contactValue[0:1] == "@"
}
