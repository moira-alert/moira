package twilio

import (
	"fmt"
	"net/url"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
)

const twimletsEchoURL = "https://twimlets.com/echo?Twiml="

type twilioSenderVoice struct {
	twilioSender
	voiceURL      string
	appendMessage bool
	twimletsEcho  bool
}

func (sender *twilioSenderVoice) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	voiceURL := sender.buildVoiceURL(trigger)

	twilioCall, err := twilio_client.NewCall(sender.client, sender.APIFromPhone, contact.Value, twilio_client.Callback(voiceURL))
	if err != nil {
		return fmt.Errorf("failed to make call to contact %s: %s", contact.Value, err.Error())
	}

	sender.logger.Debug().
		String("status", twilioCall.Status).
		String("callback_url", voiceURL).
		Msg("Call queued to twilio")

	return nil
}

func (sender *twilioSenderVoice) buildVoiceURL(trigger moira.TriggerData) string {
	message := fmt.Sprintf("Hi! This is a notification for Moira trigger %s. Please, visit Moira web interface for details.", trigger.Name)
	voiceURL := sender.voiceURL

	if sender.appendMessage {
		voiceURL += url.QueryEscape(message)
	}

	if sender.twimletsEcho {
		voiceURL = twimletsEchoURL
		voiceURL += url.QueryEscape(fmt.Sprintf("<Response><Say>%s</Say></Response>", message))
	}

	return voiceURL
}
