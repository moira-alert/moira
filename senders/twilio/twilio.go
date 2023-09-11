package twilio

import (
	"fmt"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Twilio configuration in the YAML file
type twilio struct {
	Type          string `mapstructure:"type"`
	APIAsid       string `mapstructure:"api_asid"`
	APIAuthToken  string `mapstructure:"api_authtoken"`
	APIFromPhone  string `mapstructure:"api_fromphone"`
	VoiceURL      string `mapstructure:"voiceurl"`
	TwimletsEcho  bool   `mapstructure:"twimlets_echo"`
	AppendMessage bool   `mapstructure:"append_message"`
}

// Sender implements moira sender interface via twilio
type Sender struct {
	sender sendEventsTwilio
}

type sendEventsTwilio interface {
	SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error
}

type twilioSender struct {
	client       *twilio_client.TwilioClient
	APIFromPhone string
	logger       moira.Logger
	location     *time.Location
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var t twilio
	err := mapstructure.Decode(senderSettings, &t)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to twilio config: %w", err)
	}
	apiType := t.Type

	if t.APIAsid == "" {
		return fmt.Errorf("can not read [%s] api_sid param from config", apiType)
	}

	if t.APIAuthToken == "" {
		return fmt.Errorf("can not read [%s] api_authtoken param from config", apiType)
	}

	if t.APIFromPhone == "" {
		return fmt.Errorf("can not read [%s] api_fromphone param from config", apiType)
	}

	twilioClient := twilio_client.NewClient(t.APIAsid, t.APIAuthToken)

	tSender := twilioSender{
		client:       twilioClient,
		APIFromPhone: t.APIFromPhone,
		logger:       logger,
		location:     location,
	}
	switch apiType {
	case "twilio sms":
		sender.sender = &twilioSenderSms{tSender}

	case "twilio voice":
		appendMessage := t.AppendMessage || t.TwimletsEcho

		if t.VoiceURL == "" && !t.TwimletsEcho {
			return fmt.Errorf("can not read [%s] voiceurl param from config", apiType)
		}

		sender.sender = &twilioSenderVoice{
			twilioSender:  tSender,
			voiceURL:      t.VoiceURL,
			twimletsEcho:  t.TwimletsEcho,
			appendMessage: appendMessage,
		}

	default:
		return fmt.Errorf("wrong twilio type: %s", apiType)
	}

	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	return sender.sender.SendEvents(events, contact, trigger, plots, throttled)
}
