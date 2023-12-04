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

func (notifier *StandardNotifier) registerMetrics(senderType string) {
	notifier.metrics.SendersOkMetrics.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "sends_ok")
	notifier.metrics.SendersFailedMetrics.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "sends_failed")
	notifier.metrics.SendersDroppedNotifications.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "notifications_dropped")
}

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error { //nolint
	var err error
	var sender moira.Sender
	senders := make(map[string]moira.Sender)

	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL

		senderType, ok := senderSettings["type"].(string)
		if !ok {
			return fmt.Errorf("failed to get sender type from settings")
		}

		if sender, ok = senders[senderType]; ok {
			if err = notifier.RegisterSender(senderSettings, sender); err != nil {
				return err
			}
			continue
		}

		switch senderType {
		case mailSender:
			sender = &mail.Sender{}
		case pushoverSender:
			sender = &pushover.Sender{}
		case scriptSender:
			sender = &script.Sender{}
		case discordSender:
			sender = &discord.Sender{DataBase: connector}
		case slackSender:
			sender = &slack.Sender{}
		case telegramSender:
			sender = &telegram.Sender{DataBase: connector}
		case msTeamsSender:
			sender = &msteams.Sender{}
		case pagerdutySender:
			sender = &pagerduty.Sender{ImageStores: notifier.imageStores}
		case twilioSmsSender, twilioVoiceSender:
			sender = &twilio.Sender{}
		case webhookSender:
			sender = &webhook.Sender{}
		case opsgenieSender:
			sender = &opsgenie.Sender{ImageStores: notifier.imageStores}
		case victoropsSender:
			sender = &victorops.Sender{ImageStores: notifier.imageStores}
		case mattermostSender:
			sender = &mattermost.Sender{}
		// case "email":
		// 	sender = &kontur.MailSender{}
		// case "phone":
		// 	sender = &kontur.SmsSender{}
		default:
			return fmt.Errorf("unknown sender type [%s]", senderSettings["type"])
		}

		if err = notifier.RegisterSender(senderSettings, sender); err != nil {
			return err
		}

		senders[senderType] = sender
	}

	if notifier.config.SelfStateEnabled {
		sender = &selfstate.Sender{Database: connector}
		selfStateSettings := map[string]interface{}{"type": selfStateSender}
		if err = notifier.RegisterSender(selfStateSettings, sender); err != nil {
			notifier.logger.Warning().
				Error(err).
				Msg("Failed to register selfstate sender")
		}
	}

	return nil
}

// RegisterSender adds sender for notification type and registers metrics
func (notifier *StandardNotifier) RegisterSender(senderSettings map[string]interface{}, sender moira.Sender) error {
	senderType, ok := senderSettings["type"].(string)
	if !ok {
		return fmt.Errorf("failed to retrieve sender type from sender settings")
	}

	err := sender.Init(senderSettings, notifier.logger, notifier.config.Location, notifier.config.DateTimeFormat)
	if err != nil {
		return fmt.Errorf("failed to initialize sender [%s], err [%s]", senderType, err.Error())
	}

	var senderIdent string
	if senderName, ok := senderSettings["name"]; ok {
		senderIdent, ok = senderName.(string)
		if !ok {
			return fmt.Errorf("failed to get sender name because it is not a string")
		}
	} else {
		senderIdent = senderType
	}

	notifier.sendersNameToType[senderIdent] = senderType

	if !notifier.GetSenders()[senderType] {
		eventsChannel := make(chan NotificationPackage)
		notifier.sendersNotificationsCh[senderType] = eventsChannel
		notifier.registerMetrics(senderType)
		notifier.runSenders(sender, eventsChannel)
	}

	notifier.logger.Info().
		String("sender_id", senderIdent).
		Msg("Sender registered")

	return nil
}

// GetParallelSendsPerSender returns the maximum number of running goroutines for each sentinel
func (notifier *StandardNotifier) GetMaxParallelSendsPerSender() int {
	return notifier.config.MaxParallelSendsPerSender
}

func (notifier *StandardNotifier) runSenders(sender moira.Sender, eventsChannel chan NotificationPackage) {
	for i := 0; i < notifier.GetMaxParallelSendsPerSender(); i++ {
		notifier.waitGroup.Add(1)
		go notifier.runSender(sender, eventsChannel)
	}
}

// StopSenders close all sending channels
func (notifier *StandardNotifier) StopSenders() {
	for _, ch := range notifier.sendersNotificationsCh {
		close(ch)
	}

	notifier.sendersNotificationsCh = make(map[string]chan NotificationPackage)
	notifier.logger.Info().Msg("Waiting senders finish...")
	notifier.waitGroup.Wait()
	notifier.logger.Info().Msg("Moira Notifier Senders stopped")
}

func getGraphiteSenderIdent(ident string) string {
	return strings.Replace(ident, " ", "_", -1)
}
