package notifier

import (
	"fmt"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/senders/discord"
	"github.com/moira-alert/moira/notifier/senders/mail"
	"github.com/moira-alert/moira/notifier/senders/mattermost"
	"github.com/moira-alert/moira/notifier/senders/msteams"
	"github.com/moira-alert/moira/notifier/senders/opsgenie"
	"github.com/moira-alert/moira/notifier/senders/pagerduty"
	"github.com/moira-alert/moira/notifier/senders/pushover"
	"github.com/moira-alert/moira/notifier/senders/script"
	"github.com/moira-alert/moira/notifier/senders/selfstate"
	"github.com/moira-alert/moira/notifier/senders/slack"
	"github.com/moira-alert/moira/notifier/senders/telegram"
	"github.com/moira-alert/moira/notifier/senders/twilio"
	"github.com/moira-alert/moira/notifier/senders/victorops"
	"github.com/moira-alert/moira/notifier/senders/webhook"
	// "github.com/moira-alert/moira/notifier/senders/kontur"
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

// RegisterSenders creates all senders and registers them.
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error { //nolint
	var (
		sender moira.Sender
		err    error
	)

	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL

		switch senderSettings["type"] {
		case mailSender:
			sender, err = mail.NewSender(senderSettings, notifier.logger, notifier.config.Location, notifier.config.DateTimeFormat)
		case pushoverSender:
			sender, err = pushover.NewSender(senderSettings, notifier.logger, notifier.config.Location)
		case scriptSender:
			sender, err = script.NewSender(senderSettings, notifier.logger)
		case discordSender:
			sender, err = discord.NewSender(senderSettings, notifier.logger, notifier.config.Location, connector)
		case slackSender:
			sender, err = slack.NewSender(senderSettings, notifier.logger, notifier.config.Location)
		case telegramSender:
			sender, err = telegram.NewSender(senderSettings, notifier.logger, notifier.config.Location, connector)
		case msTeamsSender:
			sender, err = msteams.NewSender(senderSettings, notifier.logger, notifier.config.Location)
		case pagerdutySender:
			sender = pagerduty.NewSender(senderSettings, notifier.logger, notifier.config.Location, notifier.imageStores)
		case twilioSmsSender, twilioVoiceSender:
			sender, err = twilio.NewSender(senderSettings, notifier.logger, notifier.config.Location)
		case webhookSender:
			sender, err = webhook.NewSender(senderSettings, notifier.logger)
		case opsgenieSender:
			sender, err = opsgenie.NewSender(senderSettings, notifier.logger, notifier.config.Location, notifier.imageStores)
		case victoropsSender:
			sender, err = victorops.NewSender(senderSettings, notifier.logger, notifier.config.Location, notifier.imageStores)
		case mattermostSender:
			sender, err = mattermost.NewSender(senderSettings, notifier.config.Location)
		// case "email":
		// sender = kontur.NewMailSender(senderSettings, notifier.logger, notifier.config.Location, notifier.config.DateTimeFormat)
		// case "phone":
		// sender = kontur.NewSmsSender(senderSettings, notifier.logger, notifier.config.Location)
		default:
			return fmt.Errorf("unknown sender type [%s]", senderSettings["type"])
		}
		if err != nil {
			return fmt.Errorf("failed to initialize sender [%s], err [%s]", senderSettings["type"], err.Error())
		}
		err = notifier.RegisterSender(senderSettings, sender)
		if err != nil {
			return fmt.Errorf("failed to register sender [%s], err [%s]", senderSettings["type"], err.Error())
		}
	}
	if notifier.config.SelfStateEnabled {
		selfStateSettings := map[string]string{"type": selfStateSender}
		sender := selfstate.NewSender(notifier.logger, connector)
		if err = notifier.RegisterSender(selfStateSettings, sender); err != nil {
			notifier.logger.Warningf("failed to register selfstate sender: %s", err.Error())
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

	eventsChannel := make(chan NotificationPackage)
	notifier.senders[senderIdent] = eventsChannel
	notifier.metrics.SendersOkMetrics.RegisterMeter(senderIdent, getGraphiteSenderIdent(senderIdent), "sends_ok")
	notifier.metrics.SendersFailedMetrics.RegisterMeter(senderIdent, getGraphiteSenderIdent(senderIdent), "sends_failed")
	notifier.runSenders(sender, eventsChannel)
	notifier.logger.Infof("Sender %s registered", senderIdent)
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
