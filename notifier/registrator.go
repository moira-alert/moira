package notifier

import (
	"fmt"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/discord"
	"github.com/moira-alert/moira/senders/mail"
	"github.com/moira-alert/moira/senders/pushover"
	"github.com/moira-alert/moira/senders/script"
	"github.com/moira-alert/moira/senders/slack"
	"github.com/moira-alert/moira/senders/telegram"
	"github.com/moira-alert/moira/senders/twilio"
	"github.com/moira-alert/moira/senders/victorops"
	"github.com/moira-alert/moira/senders/webhook"
	// "github.com/moira-alert/moira/senders/kontur"
)

const (
	mailSender        = "mail"
	pushoverSender    = "pushover"
	discordSender     = "discord"
	scriptSender      = "script"
	slackSender       = "slack"
	telegramSender    = "telegram"
	twilioSmsSender   = "twilio sms"
	twilioVoiceSender = "twilio voice"
	webhookSender     = "webhook"
	victoropsSender   = "victorops"
)

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error {
	var err error
	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL
		switch senderSettings["type"] {
		case pushoverSender:
			err = notifier.RegisterSender(senderSettings, &pushover.Sender{})
		case discordSender:
			err = notifier.RegisterSender(senderSettings, &discord.Sender{DataBase: connector})
		case slackSender:
			err = notifier.RegisterSender(senderSettings, &slack.Sender{})
		case mailSender:
			err = notifier.RegisterSender(senderSettings, &mail.Sender{})
		case telegramSender:
			err = notifier.RegisterSender(senderSettings, &telegram.Sender{DataBase: connector})
		case twilioSmsSender, twilioVoiceSender:
			err = notifier.RegisterSender(senderSettings, &twilio.Sender{})
		case scriptSender:
			err = notifier.RegisterSender(senderSettings, &script.Sender{})
		case webhookSender:
			err = notifier.RegisterSender(senderSettings, &webhook.Sender{})
		case victoropsSender:
			err = notifier.RegisterSender(senderSettings, &victorops.Sender{})
		// case "email":
		// 	err = notifier.RegisterSender(senderSettings, &kontur.MailSender{})
		// case "phone":
		// 	err = notifier.RegisterSender(senderSettings, &kontur.SmsSender{})
		default:
			return fmt.Errorf("unknown sender type [%s]", senderSettings["type"])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// RegisterSender adds sender for notification type and registers metrics
func (notifier *StandardNotifier) RegisterSender(senderSettings map[string]string, sender moira.Sender) error {
	var senderIdent string
	switch senderSettings["type"] {
	case scriptSender, webhookSender:
		senderIdent = senderSettings["name"]
	default:
		senderIdent = senderSettings["type"]
	}
	err := sender.Init(senderSettings, notifier.logger, notifier.config.Location, notifier.config.DateTimeFormat)
	if err != nil {
		return fmt.Errorf("failed to initialize sender [%s], err [%s]", senderIdent, err.Error())
	}
	ch := make(chan NotificationPackage)
	notifier.senders[senderIdent] = ch
	notifier.metrics.SendersOkMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_ok", getGraphiteSenderIdent(senderIdent)))
	notifier.metrics.SendersFailedMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_failed", getGraphiteSenderIdent(senderIdent)))
	notifier.waitGroup.Add(1)
	go notifier.run(sender, ch)
	notifier.logger.Infof("Sender %s registered", senderIdent)
	return nil
}

// StopSenders close all sending channels
func (notifier *StandardNotifier) StopSenders() {
	for _, ch := range notifier.senders {
		close(ch)
	}
	notifier.senders = make(map[string]chan NotificationPackage)
	notifier.logger.Info("Waiting senders finish...")
	notifier.waitGroup.Wait()
	notifier.logger.Info("Moira Notifier Senders stopped")
}

func getGraphiteSenderIdent(ident string) string {
	return strings.Replace(ident, " ", "_", -1)
}
