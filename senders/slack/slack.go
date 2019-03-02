package slack

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira"

	"github.com/nlopes/slack"
)

const (
	okEmoji        = ":moira-state-ok:"
	warnEmoji      = ":moira-state-warn:"
	errorEmoji     = ":moira-state-error:"
	nodataEmoji    = ":moira-state-nodata:"
	exceptionEmoji = ":moira-state-exception:"
	testEmoji      = ":moira-state-test:"

	messageMaxCharacters          = 4000
	additionalInfoCharactersCount = 400
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
	var message bytes.Buffer

	message.WriteString(fmt.Sprintf("*%s*", events.GetSubjectState()))

	tags := trigger.GetTags()
	if tags != "" {
		message.WriteString(" ")
		message.WriteString(tags)
	}

	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		message.WriteString(fmt.Sprintf(" <%s|%s>", triggerURI, trigger.Name))
	} else if trigger.Name != "" {
		message.WriteString(" ")
		message.WriteString(trigger.Name)
	}

	if trigger.Desc != "" {
		message.WriteString("\n")
		message.WriteString(trigger.Desc)
	}

	message.WriteString("\n```")

	var printEventsCount int
	messageCharsCount := len([]rune(message.String()))
	messageLimitReached := false

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		lineCharsCount := len([]rune(line))
		if messageCharsCount+lineCharsCount > messageMaxCharacters-additionalInfoCharactersCount {
			messageLimitReached = true
			break
		}
		message.WriteString(line)
		messageCharsCount += lineCharsCount
		printEventsCount++
	}
	message.WriteString("```")

	if messageLimitReached {
		message.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-printEventsCount))
	}

	if throttled {
		message.WriteString("\nPlease, *fix your system or tune this trigger* to generate less events.")
	}
	return message.String()
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
