package telegram

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
	"gopkg.in/tucnak/telebot.v2"
)

const (
	telegramLockName = "moira-telegram-users:moira-bot-host"
	workerName       = "Telebot"
	messenger        = "telegram"
	telegramLockTTL  = 30 * time.Second
)

var (
	pollerTimeout = 10 * time.Second
	emojiStates   = map[string]string{
		"OK":     "\xe2\x9c\x85",
		"WARN":   "\xe2\x9a\xa0",
		"ERROR":  "\xe2\xad\x95",
		"NODATA": "\xf0\x9f\x92\xa3",
		"TEST":   "\xf0\x9f\x98\x8a",
	}
)

// Sender implements moira sender interface via telegram
type Sender struct {
	DataBase moira.Database
	logger   moira.Logger
	apiToken string
	frontURI string
	bot      *telebot.Bot
	location *time.Location
}

// Init loads yaml config, configures and starts telegram bot
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	apiToken := senderSettings["api_token"]
	if apiToken == "" {
		return fmt.Errorf("can not read telegram api_token from config")
	}

	sender.apiToken = apiToken
	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location
	var err error
	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  sender.apiToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return err
	}

	sender.bot.Handle(telebot.OnText, func(message *telebot.Message) {
		if err = sender.handleMessage(message); err != nil {
			sender.logger.Errorf("Error handling incoming message: %s", err.Error())
		}
	})
	go sender.runTelebot()
	return nil
}

// runTelebot starts telegram bot and manages bot subscriptions
// to make sure there is always only one working Poller
func (sender *Sender) runTelebot() {
	workerAction := func(stop <-chan struct{}) error {
		sender.bot.Start()
		<-stop
		sender.bot.Stop()
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.logger,
		sender.DataBase.NewLock(telegramLockName, telegramLockTTL),
		workerAction,
	).Run(nil)
}
