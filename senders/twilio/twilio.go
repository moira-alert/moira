package twilio

import (
	"fmt"
	"time"

	twilio "github.com/carlosdp/twiliogo"
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
	client       *twilio.TwilioClient
	APIFromPhone string
	logger       moira.Logger
	location     *time.Location
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {

	apiType := senderSettings["type"]

	apiASID := senderSettings["api_asid"]
	if apiASID == "" {
		return fmt.Errorf("can not read [%s] api_sid param from config", apiType)
	}

	apiAuthToken := senderSettings["api_authtoken"]
	if apiAuthToken == "" {
		return fmt.Errorf("can not read [%s] api_authtoken param from config", apiType)
	}

	apiFromPhone := senderSettings["api_fromphone"]
	if apiFromPhone == "" {
		return fmt.Errorf("can not read [%s] api_fromphone param from config", apiType)
	}

	twilioClient := twilio.NewClient(apiASID, apiAuthToken)

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
		twimletsEcho := senderSettings["twimlets_echo"] == "true"
		appendMessage := (senderSettings["append_message"] == "true") || (twimletsEcho)

		voiceURL := senderSettings["voiceurl"]
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
