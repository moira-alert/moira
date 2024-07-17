package telegram

import (
	"errors"
	"fmt"
	"github.com/moira-alert/moira/senders/message_format"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
	"gopkg.in/tucnak/telebot.v2"
)

const (
	telegramLockPrefix = "moira-telegram-users:moira-bot-host:"
	workerName         = "Telebot"
	messenger          = "telegram"
	telegramLockTTL    = 30 * time.Second
	hidden             = "[DATA DELETED]"
)

var (
	pollerTimeout = 10 * time.Second
)

// Structure that represents the Telegram configuration in the YAML file.
type config struct {
	ContactType string `mapstructure:"contact_type"`
	APIToken    string `mapstructure:"api_token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface via telegram.
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	apiToken  string
	frontURI  string
	bot       *telebot.Bot
	location  *time.Location
	formatter message_format.MessageFormatter
}

func removeTokenFromError(err error, bot *telebot.Bot) error {
	url := telebot.DefaultApiURL
	if bot != nil {
		url = bot.URL
	}
	if err != nil && strings.Contains(err.Error(), url) {
		return errors.New(moira.ReplaceSubstring(err.Error(), "/bot", "/", hidden))
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

	emojiProvider := telegramEmojiProvider{}

	sender.apiToken = cfg.APIToken
	sender.frontURI = cfg.FrontURI
	sender.logger = logger
	sender.location = location
	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  sender.apiToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return removeTokenFromError(err, sender.bot)
	}

	sender.bot.Handle(telebot.OnText, func(message *telebot.Message) {
		if err = sender.handleMessage(message); err != nil {
			sender.logger.Error().
				Error(err).
				Msg("Error handling incoming message: %s")
		}
	})

	sender.formatter = message_format.HighlightSyntaxFormatter{
		EmojiGetter: emojiProvider,
		FrontURI:    cfg.FrontURI,
		Location:    location,
		UseEmoji:    false,
		UriFormatter: func(triggerURI, triggerName string) string {
			return fmt.Sprintf("[%s](%s)", triggerName, triggerURI)
		},
		DescriptionFormatter: func(trigger moira.TriggerData) string {
			desc := trigger.Desc
			if trigger.Desc != "" {
				desc = trigger.Desc
				desc += "\n"
			}
			return desc
		},
		BoldFormatter: func(str string) string {
			return fmt.Sprintf("**%s**", str)
		},
	}

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
