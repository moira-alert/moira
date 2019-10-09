package twilio

import (
	"bytes"
	"fmt"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
)

const printEventsCount int = 5

type twilioSenderSms struct {
	twilioSender
}

func (sender *twilioSenderSms) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	message := sender.buildMessage(events, trigger, throttled)
	sender.logger.Debugf("Calling twilio sms api to phone %s and message body %s", contact.Value, message)
	twilioMessage, err := twilio.NewMessage(sender.client, sender.APIFromPhone, contact.Value, twilio.Body(message))

	if err != nil {
		return fmt.Errorf("failed to send message to contact %s: %s", contact.Value, err)
	}

	sender.logger.Debugf(fmt.Sprintf("Message send to twilio with status: %s", twilioMessage.Status))
	return nil
}

func (sender *twilioSenderSms) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var message bytes.Buffer

	message.WriteString(fmt.Sprintf("%s %s %s (%d)\n", events.GetSubjectState(), trigger.Name, trigger.GetTags(), len(events)))
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}
		message.WriteString(fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State))
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
