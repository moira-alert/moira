package twilio

import (
	"bytes"
	"fmt"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
)

const printEventsCount int = 5

type twilioSenderSms struct {
	twilioSender
}

func (sender *twilioSenderSms) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	sender.logger.Debug().
		String("phone", contact.Value).
		String("message", message).
		Msg("Calling twilio sms api to phone %s and message body %s")

	twilioMessage, err := twilio_client.NewMessage(sender.client, sender.APIFromPhone, contact.Value, twilio_client.Body(message))
	if err != nil {
		return fmt.Errorf("failed to send message to contact %s: %w", contact.Value, err)
	}

	sender.logger.Debug().
		String("status", twilioMessage.Status).
		Msg("Message send to twilio with status")

	return nil
}

func (sender *twilioSenderSms) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message bytes.Buffer
	state := events.GetCurrentState(throttled)

	message.WriteString(fmt.Sprintf("%s %s %s (%d)\n", state, trigger.Name, trigger.GetTags(), len(events)))
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}
		message.WriteString(fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State))
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			message.WriteString(fmt.Sprintf(". %s", msg))
		}
	}

	if len(events) > printEventsCount {
		message.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-printEventsCount))
	}

	if throttled {
		message.WriteString("\n\nPlease, fix your system or tune this trigger to generate less events.")
	}
	return message.String()
}
