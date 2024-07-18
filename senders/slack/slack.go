package slack

import (
	"bytes"
	"fmt"
	"time"

	"github.com/moira-alert/moira/senders/message_format"

	"github.com/mitchellh/mapstructure"
	slackdown "github.com/moira-alert/blackfriday-slack"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/emoji_provider"

	slack_client "github.com/slack-go/slack"
)

const (
	messageMaxCharacters = 4000

	// see errors https://api.slack.com/methods/chat.postMessage
	ErrorTextChannelArchived = "is_archived"
	ErrorTextChannelNotFound = "channel_not_found"
	ErrorTextNotInChannel    = "not_in_channel"
)

// Structure that represents the Slack configuration in the YAML file.
type config struct {
	APIToken     string            `mapstructure:"api_token"`
	UseEmoji     bool              `mapstructure:"use_emoji"`
	FrontURI     string            `mapstructure:"front_uri"`
	DefaultEmoji string            `mapstructure:"default_emoji"`
	EmojiMap     map[string]string `mapstructure:"emoji_map"`
}

// Sender implements moira sender interface via slack.
type Sender struct {
	emojiProvider emoji_provider.StateEmojiGetter
	logger        moira.Logger
	client        *slack_client.Client
	formatter     message_format.MessageFormatter
}

// Init read yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to slack config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read slack api_token from config")
	}
	emojiProvider, err := emoji_provider.NewEmojiProvider(cfg.DefaultEmoji, cfg.EmojiMap)
	if err != nil {
		return fmt.Errorf("cannot initialize slack sender, err: %w", err)
	}
	sender.logger = logger
	sender.emojiProvider = emojiProvider
	sender.formatter = message_format.HighlightSyntaxFormatter{
		EmojiGetter: emojiProvider,
		FrontURI:    cfg.FrontURI,
		Location:    location,
		UseEmoji:    cfg.UseEmoji,
		UriFormatter: func(triggerURI, triggerName string) string {
			return fmt.Sprintf("<%s|%s>", triggerURI, triggerName)
		},
		DescriptionFormatter: buildDescription,
		BoldFormatter: func(str string) string {
			return fmt.Sprintf("*%s*", str)
		},
	}
	sender.client = slack_client.New(cfg.APIToken)
	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	useDirectMessaging := useDirectMessaging(contact.Value)

	state := events.GetCurrentState(throttled)
	emoji := sender.emojiProvider.GetStateEmoji(state)

	channelID, threadTimestamp, err := sender.sendMessage(message, contact.Value, trigger.ID, useDirectMessaging, emoji)
	if err != nil {
		return err
	}

	if channelID != "" && len(plots) > 0 {
		err = sender.sendPlots(plots, channelID, threadTimestamp, trigger.ID)
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
	return sender.formatter.Format(message_format.MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: messageMaxCharacters,
		Throttled:       throttled,
	})
}

func buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = string(slackdown.Run([]byte(desc)))
		desc += "\n"
	}
	return desc
}

func (sender *Sender) sendMessage(message string, contact string, triggerID string, useDirectMessaging bool, emoji string) (string, string, error) {
	params := slack_client.PostMessageParameters{
		Username:  "Moira",
		AsUser:    useDirectMessaging,
		IconEmoji: emoji,
		Markdown:  true,
		LinkNames: 1,
	}
	sender.logger.Debug().
		String("message", message).
		Msg("Calling slack")

	channelID, threadTimestamp, err := sender.client.PostMessage(contact, slack_client.MsgOptionText(message, false), slack_client.MsgOptionPostMessageParameters(params))
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

func (sender *Sender) sendPlots(plots [][]byte, channelID, threadTimestamp, triggerID string) error {
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

		_, err := sender.client.UploadFileV2(uploadParameters)
		if err != nil {
			return err
		}
	}

	return nil
}

// useDirectMessaging returns true if user contact is provided.
func useDirectMessaging(contactValue string) bool {
	return len(contactValue) > 0 && contactValue[0:1] == "@"
}
