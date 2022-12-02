package notifier

import (
	"fmt"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/discord"
	"github.com/moira-alert/moira/senders/mail"
	"github.com/moira-alert/moira/senders/mattermost"
	"github.com/moira-alert/moira/senders/msteams"
	"github.com/moira-alert/moira/senders/opsgenie"
	"github.com/moira-alert/moira/senders/pagerduty"
	"github.com/moira-alert/moira/senders/pushover"
	"github.com/moira-alert/moira/senders/script"
	"github.com/moira-alert/moira/senders/selfstate"
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
	selfStateSender   = "selfstate"
	slackSender       = "slack"
	telegramSender    = "telegram"
	twilioSmsSender   = "twilio sms"
	twilioVoiceSender = "twilio voice"
	webhookSender     = "webhook"
	opsgenieSender    = "opsgenie"
	victoropsSender   = "victorops"
	pagerdutySender   = "pagerduty"
	msTeamsSender     = "msteams"
	mattermostSender  = "mattermost"
)

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error { //nolint
	var err error
	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL
		switch senderSettings["type"] {
		case mailSender:
			err = notifier.RegisterSender(senderSettings, &mail.Sender{})
		case pushoverSender:
			err = notifier.RegisterSender(senderSettings, &pushover.Sender{})
		case scriptSender:
			err = notifier.RegisterSender(senderSettings, &script.Sender{})
		case discordSender:
			err = notifier.RegisterSender(senderSettings, &discord.Sender{DataBase: connector})
		case slackSender:
			err = notifier.RegisterSender(senderSettings, &slack.Sender{})
		case telegramSender:
			err = notifier.RegisterSender(senderSettings, &telegram.Sender{DataBase: connector})
		case msTeamsSender:
			err = notifier.RegisterSender(senderSettings, &msteams.Sender{})
		case pagerdutySender:
			err = notifier.RegisterSender(senderSettings, &pagerduty.Sender{ImageStores: notifier.imageStores})
		case twilioSmsSender, twilioVoiceSender:
			err = notifier.RegisterSender(senderSettings, &twilio.Sender{})
		case webhookSender:
			err = notifier.RegisterSender(senderSettings, &webhook.Sender{})
		case opsgenieSender:
			err = notifier.RegisterSender(senderSettings, &opsgenie.Sender{ImageStores: notifier.imageStores})
		case victoropsSender:
			err = notifier.RegisterSender(senderSettings, &victorops.Sender{ImageStores: notifier.imageStores})
		case mattermostSender:
			err = notifier.RegisterSender(senderSettings, &mattermost.Sender{})
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
	if notifier.config.SelfStateEnabled {
		selfStateSettings := map[string]string{"type": selfStateSender}
		if err = notifier.RegisterSender(selfStateSettings, &selfstate.Sender{Database: connector}); err != nil {
			notifier.logger.WarningWithError("Failed to register selfstate sender", err)
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
	eventsChannel := make(chan NotificationPackage)
	notifier.senders[senderIdent] = eventsChannel
	notifier.metrics.SendersOkMetrics.RegisterMeter(senderIdent, getGraphiteSenderIdent(senderIdent), "sends_ok")
	notifier.metrics.SendersFailedMetrics.RegisterMeter(senderIdent, getGraphiteSenderIdent(senderIdent), "sends_failed")
	notifier.runSenders(sender, eventsChannel)
	notifier.logger.Infob().
		String("sender_id", senderIdent).
		Msg("Sender registered")
	return nil
}

const maxParallelSendsPerSender = 16

func (notifier *StandardNotifier) runSenders(sender moira.Sender, eventsChannel chan NotificationPackage) {
	for i := 0; i < maxParallelSendsPerSender; i++ {
		notifier.waitGroup.Add(1)
		go notifier.runSender(sender, eventsChannel)
	}
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
