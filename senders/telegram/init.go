package telegram

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira/senders/msgformat"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
	"gopkg.in/telebot.v3"
)

const (
	telegramLockPrefix = "moira-telegram-users:moira-bot-host:"
	workerName         = "Telebot"
	messenger          = "telegram"
	telegramLockTTL    = 30 * time.Second
	hidden             = "[DATA DELETED]"
)

var (
	codeBlockStart = "<blockquote expandable>"
	codeBlockEnd   = "</blockquote>"
)

var pollerTimeout = 10 * time.Second

// Structure that represents the Telegram configuration in the YAML file.
type config struct {
	ContactType string `mapstructure:"contact_type"`
	APIToken    string `mapstructure:"api_token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Bot is abstraction over gopkg.in/telebot.v3#Bot.
type Bot interface {
	Handle(endpoint interface{}, h telebot.HandlerFunc, m ...telebot.MiddlewareFunc)
	Start()
	Stop()
	Send(to telebot.Recipient, what interface{}, opts ...interface{}) (*telebot.Message, error)
	SendAlbum(to telebot.Recipient, a telebot.Album, opts ...interface{}) ([]telebot.Message, error)
	Reply(to *telebot.Message, what interface{}, opts ...interface{}) (*telebot.Message, error)
	ChatByUsername(name string) (*telebot.Chat, error)
}

// Sender implements moira sender interface via telegram.
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	bot       Bot
	formatter msgformat.MessageFormatter
	apiToken  string
}

func (sender *Sender) removeTokenFromError(err error) error {
	if err != nil && strings.Contains(err.Error(), sender.apiToken) {
		return errors.New(strings.Replace(err.Error(), sender.apiToken, hidden, -1))
	}
	return err
}

// Init loads yaml config, configures and starts telegram bot.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to telegram config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read telegram api_token from config")
	}
	sender.apiToken = cfg.APIToken

	emojiProvider := telegramEmojiProvider{}
	sender.formatter = msgformat.NewHighlightSyntaxFormatter(
		emojiProvider,
		true,
		cfg.FrontURI,
		location,
		urlFormatter,
		emptyDescriptionFormatter,
		boldFormatter,
		eventStringFormatter,
		codeBlockStart,
		codeBlockEnd)

	sender.logger = logger
	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  cfg.APIToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return sender.removeTokenFromError(err)
	}

	sender.bot.Handle(telebot.OnText, func(ctx telebot.Context) error {
		if err = sender.handleMessage(ctx.Message()); err != nil {
			sender.logger.Error().
				Error(err).
				Msg("Error handling incoming message")
			return err
		}
		return nil
	})

	go sender.runTelebot(cfg.ContactType)

	return nil
}

// runTelebot starts telegram bot and manages bot subscriptions
// to make sure there is always only one working Poller.
func (sender *Sender) runTelebot(contactType string) {
	workerAction := func(stop <-chan struct{}) error {
		sender.bot.Start()
		<-stop
		sender.bot.Stop()
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.logger,
		sender.DataBase.NewLock(telegramLockKey(contactType), telegramLockTTL),
		workerAction,
	).Run(nil)
}

func telegramLockKey(contactType string) string {
	return telegramLockPrefix + contactType
}

func urlFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", triggerURI, triggerName)
}

func emptyDescriptionFormatter(trigger moira.TriggerData) string {
	return ""
}

func boldFormatter(str string) string {
	return fmt.Sprintf("<b>%s</b>", str)
}

func eventStringFormatter(event moira.NotificationEvent, loc *time.Location) string {
	return fmt.Sprintf(
		"%s: <code>%s</code> = %s (%s to %s)",
		event.FormatTimestamp(loc, moira.DefaultTimeFormat),
		event.Metric,
		event.GetMetricsValues(moira.DefaultNotificationSettings),
		event.OldState,
		event.State)
}
