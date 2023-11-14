package telegram

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
	"gopkg.in/tucnak/telebot.v2"
)

const (
	telegramLockName = "moira-telegram-users:moira-bot-host:"
	workerName       = "Telebot"
	messenger        = "telegram"
	telegramLockTTL  = 30 * time.Second
	hidden           = "[DATA DELETED]"
)

var (
	pollerTimeout = 10 * time.Second
	emojiStates   = map[moira.State]string{
		moira.StateOK:     "\xe2\x9c\x85",
		moira.StateWARN:   "\xe2\x9a\xa0",
		moira.StateERROR:  "\xe2\xad\x95",
		moira.StateNODATA: "\xf0\x9f\x92\xa3",
		moira.StateTEST:   "\xf0\x9f\x98\x8a",
	}
)

// Structure that represents the Telegram configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	APIToken string `mapstructure:"api_token"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface via telegram
type Sender struct {
	clients map[string]*telegramClient
}

type telegramClient struct {
	database moira.Database
	logger   moira.Logger
	apiToken string
	frontURI string
	bot      *telebot.Bot
	location *time.Location
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

// Init loads yaml config, configures and starts telegram bot
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to telegram config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read telegram api_token from config")
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.APIToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return removeTokenFromError(err, bot)
	}

	client := &telegramClient{
		apiToken: cfg.APIToken,
		frontURI: cfg.FrontURI,
		logger:   opts.Logger,
		location: opts.Location,
		bot:      bot,
		database: opts.Database,
	}

	client.bot.Handle(telebot.OnText, func(message *telebot.Message) {
		if err = client.handleMessage(message); err != nil {
			client.logger.Error().
				Error(err).
				Msg("Error handling incoming message: %s")
		}
	})

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	go client.runTelebot(senderIdent)

	if sender.clients == nil {
		sender.clients = make(map[string]*telegramClient)
	}

	sender.clients[senderIdent] = client

	return nil
}

// runTelebot starts telegram bot and manages bot subscriptions
// to make sure there is always only one working Poller
func (client *telegramClient) runTelebot(senderIdent string) {
	workerAction := func(stop <-chan struct{}) error {
		client.bot.Start()
		<-stop
		client.bot.Stop()
		return nil
	}

	worker.NewWorker(
		workerName,
		client.logger,
		client.database.NewLock(telegramLockName+senderIdent, telegramLockTTL),
		workerAction,
	).Run(nil)
}
