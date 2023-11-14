package notifier

import (
	"fmt"
	"strings"
	"sync"

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
func (notifier *StandardNotifier) RegisterSenders() error { //nolint
	var err error
	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL

		senderType, ok := senderSettings["type"].(string)
		if !ok {
			return fmt.Errorf("failed to get sender type from settings")
		}

		if sender, ok := notifier.senders[senderType]; ok {
			if err = notifier.RegisterSender(senderSettings, sender); err != nil {
				return err
			}
		}

		switch senderType {
		case mailSender:
			err = notifier.RegisterSender(senderSettings, &mail.Sender{})
		case pushoverSender:
			err = notifier.RegisterSender(senderSettings, &pushover.Sender{})
		case scriptSender:
			err = notifier.RegisterSender(senderSettings, &script.Sender{})
		case discordSender:
			err = notifier.RegisterSender(senderSettings, &discord.Sender{})
		case slackSender:
			err = notifier.RegisterSender(senderSettings, &slack.Sender{})
		case telegramSender:
			err = notifier.RegisterSender(senderSettings, &telegram.Sender{})
		case msTeamsSender:
			err = notifier.RegisterSender(senderSettings, &msteams.Sender{})
		case pagerdutySender:
			err = notifier.RegisterSender(senderSettings, &pagerduty.Sender{})
		case twilioSmsSender, twilioVoiceSender:
			err = notifier.RegisterSender(senderSettings, &twilio.Sender{})
		case webhookSender:
			err = notifier.RegisterSender(senderSettings, &webhook.Sender{})
		case opsgenieSender:
			err = notifier.RegisterSender(senderSettings, &opsgenie.Sender{})
		case victoropsSender:
			err = notifier.RegisterSender(senderSettings, &victorops.Sender{})
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
		selfStateSettings := map[string]interface{}{"type": selfStateSender}
		if err = notifier.RegisterSender(selfStateSettings, &selfstate.Sender{}); err != nil {
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

	if _, ok := notifier.sendersOnce[senderType]; !ok {
		notifier.sendersOnce[senderType] = &sync.Once{}
	}

	opts := moira.InitOptions{
		SenderSettings: senderSettings,
		Logger:         notifier.logger,
		Location:       notifier.config.Location,
		DateTimeFormat: notifier.config.DateTimeFormat,
		Database:       notifier.database,
		ImageStores:    notifier.imageStores,
	}

	err := sender.Init(opts)
	if err != nil {
		return fmt.Errorf("failed to initialize sender [%s], err [%s]", senderType, err.Error())
	}

	notifier.sendersOnce[senderType].Do(func() {
		eventsChannel := make(chan NotificationPackage)
		notifier.sendersNotificationsCh[senderType] = eventsChannel

		notifier.metrics.SendersOkMetrics.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "sends_ok")
		notifier.metrics.SendersFailedMetrics.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "sends_failed")
		notifier.metrics.SendersDroppedNotifications.RegisterMeter(senderType, getGraphiteSenderIdent(senderType), "notifications_dropped")
		notifier.runSenders(sender, eventsChannel)
	})

	notifier.senders[senderType] = sender

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
