package twilio

import (
	"fmt"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Twilio configuration in the YAML file.
type config struct {
	Type          string `mapstructure:"sender_type" validate:"required"`
	APIAsid       string `mapstructure:"api_asid" validate:"required"`
	APIAuthToken  string `mapstructure:"api_authtoken" validate:"required"`
	APIFromPhone  string `mapstructure:"api_fromphone" validate:"required"`
	VoiceURL      string `mapstructure:"voiceurl"`
	TwimletsEcho  bool   `mapstructure:"twimlets_echo"`
	AppendMessage bool   `mapstructure:"append_message"`
}

// Sender implements moira sender interface via twilio.
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

// Init read yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config

	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to twilio config: %w", err)
	}

	if err = moira.ValidateStruct(cfg); err != nil {
		return fmt.Errorf("twilio config validation error: %w", err)
	}

	apiType := cfg.Type

	twilioClient := twilio_client.NewClient(cfg.APIAsid, cfg.APIAuthToken)

	tSender := twilioSender{
		client:       twilioClient,
		APIFromPhone: cfg.APIFromPhone,
		logger:       logger,
		location:     location,
	}

	switch apiType {
	case "twilio sms":
		sender.sender = &twilioSenderSms{tSender}

	case "twilio voice":
		appendMessage := cfg.AppendMessage || cfg.TwimletsEcho

		if cfg.VoiceURL == "" && !cfg.TwimletsEcho {
			return fmt.Errorf("can not read [%s] voiceurl param from config", apiType)
		}

		sender.sender = &twilioSenderVoice{
			twilioSender:  tSender,
			voiceURL:      cfg.VoiceURL,
			twimletsEcho:  cfg.TwimletsEcho,
			appendMessage: appendMessage,
		}

	default:
		return fmt.Errorf("wrong twilio type: %s", apiType)
	}

	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	return sender.sender.SendEvents(events, contact, trigger, plots, throttled)
}
