package telegram

import (
	"fmt"
	"time"

	"gopkg.in/tucnak/telebot.v2"

	"github.com/moira-alert/moira"
)

const telegramLockName = "moira-telegram-users:moira-bot-host"
const messenger = "telegram"

var (
	telegramMessageLimit    = 4096
	pollerTimeout           = 10 * time.Second
	telegramLockTTL         = 30 * time.Second
	singlePollerStateExpiry = time.Minute
	emojiStates             = map[string]string{
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
	APIToken string
	FrontURI string
	logger   moira.Logger
	bot      *telebot.Bot
	location *time.Location
}

// Init loads yaml config, configures and starts telegram bot
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var err error
	sender.APIToken = senderSettings["api_token"]
	if sender.APIToken == "" {
		return fmt.Errorf("can not read telegram api_token from config")
	}
	sender.FrontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location

	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  sender.APIToken,
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
	lock := sender.DataBase.NewLock(telegramLockName, telegramLockTTL)
	for {
		lost, err := lock.Acquire(nil)
		if err != nil {
			continue
		}
		go sender.bot.Start()
		<-lost
		sender.bot.Stop()
		lock.Release()
	}
}
