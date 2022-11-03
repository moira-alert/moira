package telegram

import (
	"fmt"
	"strconv"
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
	defaultPollerTimeout = 10 * time.Second
	emojiStates          = map[moira.State]string{
		moira.StateOK:     "\xe2\x9c\x85",
		moira.StateWARN:   "\xe2\x9a\xa0",
		moira.StateERROR:  "\xe2\xad\x95",
		moira.StateNODATA: "\xf0\x9f\x92\xa3",
		moira.StateTEST:   "\xf0\x9f\x98\x8a",
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
		Poller: &telebot.LongPoller{Timeout: sender.getPollerTimeout(senderSettings["poller_timeout_sec"])},
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

func (sender *Sender) getPollerTimeout(pollerTimeoutSecStr string) time.Duration {
	if pollerTimeoutSecStr == "" {
		sender.logger.Infof("Not set poller timeout, use default")

		return defaultPollerTimeout
	}

	pollerTimeoutSec, err := strconv.Atoi(pollerTimeoutSecStr)
	if err != nil {
		sender.logger.Warningf("Error cast poller timeout : %s. Will use default value", err.Error())

		return defaultPollerTimeout
	}

	return time.Duration(pollerTimeoutSec) * time.Second
}
