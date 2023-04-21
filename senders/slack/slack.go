package slack

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/moira-alert/moira/senders/default_sender"

	"github.com/moira-alert/moira"
	"github.com/slack-go/slack"
)

const (
	okEmoji        = ":moira-state-ok:"
	warnEmoji      = ":moira-state-warn:"
	errorEmoji     = ":moira-state-error:"
	nodataEmoji    = ":moira-state-nodata:"
	exceptionEmoji = ":moira-state-exception:"
	testEmoji      = ":moira-state-test:"

	messageMaxCharacters = 10000

	//see errors https://api.slack.com/methods/chat.postMessage
	errorTextChannelArchived = "is_archived"
	errorTextChannelNotFound = "channel_not_found"
	errorTextNotInChannel    = "not_in_channel"
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
	useEmoji bool
	client   *slack.Client
	*default_sender.DefaultSender
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	apiToken := senderSettings["api_token"]
	if apiToken == "" {
		return fmt.Errorf("can not read slack api_token from config")
	}
	sender.useEmoji, _ = strconv.ParseBool(senderSettings["use_emoji"])
	sender.DefaultSender = default_sender.NewDefaultSender(
		senderSettings["front_uri"],
		messageMaxCharacters,
		logger,
		location,
	)
	sender.client = slack.New(apiToken)
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.BuildMessage(events, trigger, throttled)
	useDirectMessaging := useDirectMessaging(contact.Value)
	emoji := sender.getStateEmoji(events.GetSubjectState())
	channelID, threadTimestamp, err := sender.sendMessage(message, contact.Value, trigger.ID, useDirectMessaging, emoji)
	if err != nil {
		return err
	}
	if channelID != "" && len(plots) > 0 {
		err = sender.sendPlots(plots, channelID, threadTimestamp, trigger.ID)
		if err != nil {
			sender.Logger.Warning().
				String("trigger_id", trigger.ID).
				String("contact_value", contact.Value).
				String("contact_type", contact.Type).
				Error(err)
		}
	}
	return nil
}

func (sender *Sender) sendMessage(message string, contact string, triggerID string, useDirectMessaging bool, emoji string) (string, string, error) {
	params := slack.PostMessageParameters{
		Username:  "Moira",
		AsUser:    useDirectMessaging,
		IconEmoji: emoji,
		Markdown:  true,
		LinkNames: 1,
	}
	sender.Logger.Debug().
		String("message", message).
		Msg("Calling slack")

	channelID, threadTimestamp, err := sender.client.PostMessage(contact, slack.MsgOptionText(message, false), slack.MsgOptionPostMessageParameters(params))
	if err != nil {
		errorText := err.Error()
		if errorText == errorTextChannelArchived || errorText == errorTextNotInChannel ||
			errorText == errorTextChannelNotFound {
			return channelID, threadTimestamp, moira.NewSenderBrokenContactError(err)
		}
		return channelID, threadTimestamp, fmt.Errorf("failed to send %s event message to slack [%s]: %s",
			triggerID, contact, errorText)
	}
	return channelID, threadTimestamp, nil
}

func (sender *Sender) sendPlots(plots [][]byte, channelID, threadTimestamp, triggerID string) error {
	filename := fmt.Sprintf("%s.png", triggerID)
	for _, plot := range plots {
		reader := bytes.NewReader(plot)
		uploadParameters := slack.UploadFileV2Parameters{
			FileSize:        len(plot),
			Reader:          reader,
			Title:           filename,
			Filename:        filename,
			Channel:         channelID,
			ThreadTimestamp: threadTimestamp,
		}

		_, err := sender.client.UploadFileV2(uploadParameters)
		if err != nil {
			return err
		}
	}

	return nil
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
