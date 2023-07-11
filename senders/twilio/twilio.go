package twilio

import (
	"fmt"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

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

// Structure that represents the Twilio configuration in the YAML file
type Twilio struct {
	Type          string `mapstructure:"type"`
	APIAsid       string `mapstructure:"api_asid"`
	APIAuthToken  string `mapstructure:"api_authtoken"`
	APIFromPhone  string `mapstructure:"api_fromphone"`
	VoiceURL      string `mapstructure:"voiceurl"`
	TwimletsEcho  string `mapstructure:"twimlets_echo"`
	AppendMessage string `mapstructure:"append_message"`
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var twilio Twilio
	err := mapstructure.Decode(senderSettings, &twilio)
	if err != nil {
		return fmt.Errorf("decoding error from yaml file to twilio structure: %s", err)
	}
	apiType := twilio.Type

	apiASID := twilio.APIAsid
	if apiASID == "" {
		return fmt.Errorf("can not read [%s] api_sid param from config", apiType)
	}

	apiAuthToken := twilio.APIAuthToken
	if apiAuthToken == "" {
		return fmt.Errorf("can not read [%s] api_authtoken param from config", apiType)
	}

	apiFromPhone := twilio.APIFromPhone
	if apiFromPhone == "" {
		return fmt.Errorf("can not read [%s] api_fromphone param from config", apiType)
	}

	twilioClient := twilio_client.NewClient(apiASID, apiAuthToken)

	twilioSender1 := twilioSender{
		client:       twilioClient,
		APIFromPhone: apiFromPhone,
		logger:       logger,
		location:     location,
	}
	switch apiType {
	case "twilio sms":
		sender.sender = &twilioSenderSms{twilioSender1}

	case "twilio voice":
		twimletsEcho := twilio.TwimletsEcho == "true" //nolint
		appendMessage := (twilio.AppendMessage == "true") || (twimletsEcho)

		voiceURL := twilio.VoiceURL
		if voiceURL == "" && !twimletsEcho {
			return fmt.Errorf("can not read [%s] voiceurl param from config", apiType)
		}

		sender.sender = &twilioSenderVoice{
			twilioSender:  twilioSender1,
			voiceURL:      voiceURL,
			twimletsEcho:  twimletsEcho,
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
