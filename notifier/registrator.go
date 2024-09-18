package notifier

import (
	"errors"
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

var (
	ErrSenderRegistered   = errors.New("sender is already registered")
	ErrMissingSenderType  = errors.New("failed to retrieve sender type from sender settings")
	ErrMissingContactType = errors.New("failed to retrieve sender contact type from sender settings")
)

// RegisterSenders watch on senders config and register all configured senders.
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database) error { //nolint
	var err error
	for _, senderSettings := range notifier.config.Senders {
		senderSettings["front_uri"] = notifier.config.FrontURL
		switch senderSettings["sender_type"] {
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
			return fmt.Errorf("unknown sender type [%s]", senderSettings["sender_type"])
		}

		if err != nil {
			return err
		}
	}
	if notifier.config.SelfStateEnabled {
		selfStateSettings := map[string]interface{}{
			"sender_type":  selfStateSender,
			"contact_type": selfStateSender,
		}
		if err = notifier.RegisterSender(selfStateSettings, &selfstate.Sender{Database: connector}); err != nil {
			notifier.logger.Warning().
				Error(err).
				Msg("Failed to register selfstate sender")
		}
	}

	return nil
}

func (notifier *StandardNotifier) registerMetrics(senderContactType string) {
	notifier.metrics.SendersOkMetrics.RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_ok")
	notifier.metrics.SendersFailedMetrics.RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "sends_failed")
	notifier.metrics.SendersDroppedNotifications.RegisterMeter(senderContactType, getGraphiteSenderIdent(senderContactType), "notifications_dropped")
}

// RegisterSender adds sender for notification type and registers metrics.
func (notifier *StandardNotifier) RegisterSender(senderSettings map[string]interface{}, sender moira.Sender) error {
	senderType, ok := senderSettings["sender_type"].(string)
	if !ok {
		return ErrMissingSenderType
	}

	senderContactType, ok := senderSettings["contact_type"].(string)
	if !ok {
		return ErrMissingContactType
	}

	if _, ok := notifier.senders[senderContactType]; ok {
		return fmt.Errorf("failed to initialize sender [%s], err [%w]", senderContactType, ErrSenderRegistered)
	}

	err := sender.Init(senderSettings, notifier.logger, notifier.config.Location, notifier.config.DateTimeFormat)
	if err != nil {
		return fmt.Errorf("failed to initialize sender [%s], err [%w]", senderContactType, err)
	}

	eventsChannel := make(chan NotificationPackage)
	notifier.senders[senderContactType] = eventsChannel

	notifier.registerMetrics(senderContactType)
	notifier.runSenders(sender, eventsChannel)

	notifier.logger.Info().
		String("sender_contact_type", senderContactType).
		String("sender_type", senderType).
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

// StopSenders close all sending channels.
func (notifier *StandardNotifier) StopSenders() {
	for _, ch := range notifier.senders {
		close(ch)
	}
	notifier.senders = make(map[string]chan NotificationPackage)
	notifier.logger.Info().Msg("Waiting senders finish...")
	notifier.waitGroup.Wait()
	notifier.logger.Info().Msg("Moira Notifier Senders stopped")
}

func getGraphiteSenderIdent(ident string) string {
	return strings.ReplaceAll(ident, " ", "_")
}
