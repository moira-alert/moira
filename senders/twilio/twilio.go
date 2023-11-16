package twilio

import (
	"fmt"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Twilio configuration in the YAML file
type config struct {
	Type          string `mapstructure:"type"`
	Name          string `mapstructure:"name"`
	APIAsid       string `mapstructure:"api_asid"`
	APIAuthToken  string `mapstructure:"api_authtoken"`
	APIFromPhone  string `mapstructure:"api_fromphone"`
	VoiceURL      string `mapstructure:"voiceurl"`
	TwimletsEcho  bool   `mapstructure:"twimlets_echo"`
	AppendMessage bool   `mapstructure:"append_message"`
}

// Sender implements moira sender interface via twilio
type Sender struct {
	clients map[string]sendEventsTwilio
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
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to twilio config: %w", err)
	}

	apiType := cfg.Type

	if cfg.APIAsid == "" {
		return fmt.Errorf("can not read [%s] api_sid param from config", apiType)
	}

	if cfg.APIAuthToken == "" {
		return fmt.Errorf("can not read [%s] api_authtoken param from config", apiType)
	}

	if cfg.APIFromPhone == "" {
		return fmt.Errorf("can not read [%s] api_fromphone param from config", apiType)
	}

	twilioClient := twilio_client.NewClient(cfg.APIAsid, cfg.APIAuthToken)

	tSender := twilioSender{
		client:       twilioClient,
		APIFromPhone: cfg.APIFromPhone,
		logger:       opts.Logger,
		location:     opts.Location,
	}

	var twilioSender sendEventsTwilio

	switch apiType {
	case "twilio sms":
		twilioSender = &twilioSenderSms{tSender}

	case "twilio voice":
		appendMessage := cfg.AppendMessage || cfg.TwimletsEcho

		if cfg.VoiceURL == "" && !cfg.TwimletsEcho {
			return fmt.Errorf("can not read [%s] voiceurl param from config", apiType)
		}

		twilioSender = &twilioSenderVoice{
			twilioSender:  tSender,
			voiceURL:      cfg.VoiceURL,
			twimletsEcho:  cfg.TwimletsEcho,
			appendMessage: appendMessage,
		}

	default:
		return fmt.Errorf("wrong twilio type: %s", apiType)
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]sendEventsTwilio)
	}

	sender.clients[senderIdent] = twilioSender

	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	twilioClient, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	return twilioClient.SendEvents(events, contact, trigger, plots, throttled)
}
