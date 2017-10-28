package notifier

import (
	"fmt"
	"strings"

	// "git.skbkontur.ru/devops/kontur"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/mail"
	"github.com/moira-alert/moira/senders/pushover"
	"github.com/moira-alert/moira/senders/script"
	"github.com/moira-alert/moira/senders/slack"
	"github.com/moira-alert/moira/senders/telegram"
	"github.com/moira-alert/moira/senders/twilio"
)

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error {
	for _, senderSettings := range notifier.config.Senders {
		notifier.logger.Infof("Registering sender %s", senderSettings["type"])
		senderSettings["front_uri"] = notifier.config.FrontURL
		switch senderSettings["type"] {
		case "pushover":
			if err := notifier.RegisterSender(senderSettings, &pushover.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "slack":
			if err := notifier.RegisterSender(senderSettings, &slack.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "mail":
			if err := notifier.RegisterSender(senderSettings, &mail.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "script":
			if err := notifier.RegisterSender(senderSettings, &script.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "telegram":
			if err := notifier.RegisterSender(senderSettings, &telegram.Sender{DataBase: connector}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "twilio sms":
			if err := notifier.RegisterSender(senderSettings, &twilio.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		case "twilio voice":
			if err := notifier.RegisterSender(senderSettings, &twilio.Sender{}); err != nil {
				notifier.logger.Fatalf("Can not register sender %s: %s", senderSettings["type"], err)
			}
		// case "email":
		// 	if err := notifier.RegisterSender(senderSettings, &kontur.MailSender{}); err != nil {
		// 	}
		// case "phone":
		// 	if err := notifier.RegisterSender(senderSettings, &kontur.SmsSender{}); err != nil {
		// 	}
		default:
			return fmt.Errorf("Unknown sender type [%s]", senderSettings["type"])
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
	err := sender.Init(senderSettings, notifier.logger, notifier.config.Location)
	if err != nil {
		return fmt.Errorf("Don't initialize sender [%s], err [%s]", senderIdent, err.Error())
	}
	ch := make(chan NotificationPackage)
	notifier.senders[senderIdent] = ch
	notifier.metrics.SendersOkMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_ok", getGraphiteSenderIdent(senderIdent)))
	notifier.metrics.SendersFailedMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_failed", getGraphiteSenderIdent(senderIdent)))
	notifier.waitGroup.Add(1)
	go notifier.run(sender, ch)
	notifier.logger.Debugf("Sender %s registered", senderIdent)
	return nil
}

// StopSenders close all sending channels
func (notifier *StandardNotifier) StopSenders() {
	for _, ch := range notifier.senders {
		close(ch)
	}
	notifier.senders = make(map[string]chan NotificationPackage)
	notifier.logger.Debug("Waiting senders finish ...")
	notifier.waitGroup.Wait()
}

func getGraphiteSenderIdent(ident string) string {
	return strings.Replace(ident, " ", "_", -1)
}
