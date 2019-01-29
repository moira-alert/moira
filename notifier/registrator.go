package notifier

import (
	"fmt"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/mail"
	"github.com/moira-alert/moira/senders/pushover"
	"github.com/moira-alert/moira/senders/script"
	"github.com/moira-alert/moira/senders/slack"
	"github.com/moira-alert/moira/senders/telegram"
	"github.com/moira-alert/moira/senders/twilio"
	"github.com/moira-alert/moira/senders/webhook"
	// "github.com/moira-alert/moira/senders/kontur"
)

const (
	mailSender        = "mail"
	pushoverSender    = "pushover"
	scriptSender      = "script"
	slackSender       = "slack"
	telegramSender    = "telegram"
	twilioSmsSender   = "twilio sms"
	twilioVoiceSender = "twilio voice"
	webhookSender     = "webhook"
)

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error {
	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL
		switch senderSettings["type"] {
		case pushoverSender:
			return notifier.RegisterSender(senderSettings, &pushover.Sender{})
		case slackSender:
			return notifier.RegisterSender(senderSettings, &slack.Sender{})
		case mailSender:
			return notifier.RegisterSender(senderSettings, &mail.Sender{})
		case telegramSender:
			return notifier.RegisterSender(senderSettings, &telegram.Sender{DataBase: connector})
		case twilioSmsSender, twilioVoiceSender:
			return notifier.RegisterSender(senderSettings, &twilio.Sender{})
		case scriptSender:
			return notifier.RegisterSender(senderSettings, &script.Sender{})
		case webhookSender:
			return notifier.RegisterSender(senderSettings, &webhook.Sender{})
		// case "email":
		// 	return notifier.RegisterSender(senderSettings, &kontur.MailSender{})
		// case "phone":
		// 	return notifier.RegisterSender(senderSettings, &kontur.SmsSender{})
		default:
			return fmt.Errorf("unknown sender type [%s]", senderSettings["type"])
		}
	}
	return nil
}

// RegisterSender adds sender for notification type and registers metrics
func (notifier *StandardNotifier) RegisterSender(senderSettings map[string]string, sender moira.Sender) error {
	var senderIdent string
	if senderSettings["type"] == "script" {
		senderIdent = senderSettings["name"]
	} else {
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
