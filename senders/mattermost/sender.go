package mattermost

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/moira-alert/moira/senders/msgformat"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/emoji_provider"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mitchellh/mapstructure"
)

// Structure that represents the Mattermost configuration in the YAML file.
type config struct {
	Url          string            `mapstructure:"url"`
	InsecureTLS  bool              `mapstructure:"insecure_tls"`
	APIToken     string            `mapstructure:"api_token"`
	FrontURI     string            `mapstructure:"front_uri"`
	UseEmoji     bool              `mapstructure:"use_emoji"`
	DefaultEmoji string            `mapstructure:"default_emoji"`
	EmojiMap     map[string]string `mapstructure:"emoji_map"`
}

// Sender posts messages to Mattermost chat.
// It implements moira.Sender.
// You must call Init method before SendEvents method.
type Sender struct {
	logger    moira.Logger
	client    Client
	formatter msgformat.MessageFormatter
}

const (
	messageMaxCharacters = 4_000
)

var (
	codeBlockStart = "```"
	codeBlockEnd   = "```"
)

// Init configures Sender.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, _ string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to mattermost config: %w", err)
	}

	if cfg.Url == "" {
		return fmt.Errorf("can not read Mattermost url from config")
	}

	client := model.NewAPIv4Client(cfg.Url)

	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureTLS,
			},
		},
	}

	sender.client = client

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read Mattermost api_token from config")
	}
	sender.client.SetToken(cfg.APIToken)

	if cfg.FrontURI == "" {
		return fmt.Errorf("can not read Mattermost front_uri from config")
	}

	emojiProvider, err := emoji_provider.NewEmojiProvider(cfg.DefaultEmoji, cfg.EmojiMap)
	if err != nil {
		return fmt.Errorf("cannot initialize mattermost sender, err: %w", err)
	}
	sender.logger = logger
	sender.formatter = msgformat.NewHighlightSyntaxFormatter(
		emojiProvider,
		cfg.UseEmoji,
		cfg.FrontURI,
		location,
		uriFormatter,
		descriptionFormatter,
		descriptionCutter,
		boldFormatter,
		eventStringFormatter,
		codeBlockStart,
		codeBlockEnd)

	return nil
}

func uriFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("[%s](%s)", triggerName, triggerURI)
}

func descriptionFormatter(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	}
	return desc
}

func descriptionCutter(desc string, maxSize int) string {
	return desc[:maxSize] + "...\n"
}

func boldFormatter(str string) string {
	return fmt.Sprintf("**%s**", str)
}

func eventStringFormatter(event moira.NotificationEvent, loc *time.Location) string {
	return fmt.Sprintf(
		"%s: %s = %s (%s to %s)",
		event.FormatTimestamp(loc, moira.DefaultTimeFormat),
		event.Metric,
		event.GetMetricsValues(moira.DefaultNotificationSettings),
		event.OldState,
		event.State)
}

// SendEvents implements moira.Sender interface.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	ctx := context.Background()
	post, err := sender.sendMessage(ctx, message, contact.Value, trigger.ID)
	if err != nil {
		return err
	}
	if len(plots) > 0 {
		err = sender.sendPlots(ctx, plots, contact.Value, post.Id, trigger.ID)
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
	return sender.formatter.Format(msgformat.MessageFormatterParams{
		Events:          events,
		Trigger:         trigger,
		MessageMaxChars: messageMaxCharacters,
		Throttled:       throttled,
	})
}

func (sender *Sender) sendMessage(ctx context.Context, message string, contact string, triggerID string) (*model.Post, error) {
	post := model.Post{
		ChannelId: contact,
		Message:   message,
	}

	sentPost, _, err := sender.client.CreatePost(ctx, &post)
	if err != nil {
		return nil, fmt.Errorf("failed to send %s event message to Mattermost [%s]: %w", triggerID, contact, err)
	}

	return sentPost, nil
}

func (sender *Sender) sendPlots(ctx context.Context, plots [][]byte, channelID, postID, triggerID string) error {
	var filesID []string

	filename := fmt.Sprintf("%s.png", triggerID)
	for _, plot := range plots {
		file, _, err := sender.client.UploadFile(ctx, plot, channelID, filename)
		if err != nil {
			return err
		}
		for _, info := range file.FileInfos {
			filesID = append(filesID, info.Id)
		}
	}

	if len(filesID) > 0 {
		_, _, err := sender.client.CreatePost(
			ctx,
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
