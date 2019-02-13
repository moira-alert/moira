package twilio

import (
	"fmt"
	"net/url"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
)

type twilioSenderVoice struct {
	twilioSender
	voiceURL      string
	appendMessage bool
}

func (sender *twilioSenderVoice) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	voiceURL := sender.buildVoiceURL(trigger)
	twilioCall, err := twilio.NewCall(sender.client, sender.APIFromPhone, contact.Value, twilio.Callback(voiceURL))
	if err != nil {
		return fmt.Errorf("failed to make call to contact %s: %s", contact.Value, err.Error())
	}
	sender.logger.Debugf("Call queued to twilio with status %s, callback url %s", twilioCall.Status, voiceURL)
	return nil
}

func (sender *twilioSenderVoice) buildVoiceURL(trigger moira.TriggerData) string {
	voiceURL := sender.voiceURL
	if sender.appendMessage {
		voiceURL += url.QueryEscape(fmt.Sprintf("Hi! This is a notification for Moira trigger %s. Please, visit Moira web interface for details.", trigger.Name))
	}
	return voiceURL
}
