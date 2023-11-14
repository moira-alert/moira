package pushover

import (
	"bytes"
	"fmt"
	"time"

	"github.com/moira-alert/moira"

	pushover_client "github.com/gregdel/pushover"
	"github.com/mitchellh/mapstructure"
)

const printEventsCount int = 5
const titleLimit = 250
const urlLimit = 512

// Structure that represents the Pushover configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	APIToken string `mapstructure:"api_token"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface via pushover
type Sender struct {
	clients map[string]*pushoverClient
}

type pushoverClient struct {
	logger   moira.Logger
	location *time.Location
	client   *pushover_client.Pushover

	apiToken string
	frontURI string
}

// Init read yaml config
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to pushover config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read pushover api_token from config")
	}

	client := &pushoverClient{
		apiToken: cfg.APIToken,
		client:   pushover_client.New(cfg.APIToken),
		logger:   opts.Logger,
		frontURI: cfg.FrontURI,
		location: opts.Location,
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*pushoverClient)
	}

	sender.clients[senderIdent] = client

	return nil
}

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	pushoverClient, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	pushoverMessage := pushoverClient.makePushoverMessage(events, trigger, plots, throttled)

	pushoverClient.logger.Debug().
		String("message_title", pushoverMessage.Title).
		String("message", pushoverMessage.Message).
		Msg("Calling pushover with message title")

	recipient := pushover_client.NewRecipient(contact.Value)
	_, err := pushoverClient.client.SendMessage(pushoverMessage, recipient)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
	}

	return nil
}

func (client *pushoverClient) makePushoverMessage(events moira.NotificationEvents, trigger moira.TriggerData, plots [][]byte, throttled bool) *pushover_client.Message {
	pushoverMessage := &pushover_client.Message{
		Message:   client.buildMessage(events, throttled),
		Title:     client.buildTitle(events, trigger, throttled),
		Priority:  client.getMessagePriority(events),
		Retry:     5 * time.Minute, //nolint
		Expire:    time.Hour,
		Timestamp: events[len(events)-1].Timestamp,
	}

	url := trigger.GetTriggerURI(client.frontURI)
	if len(url) < urlLimit {
		pushoverMessage.URL = url
	}

	if len(plots) > 0 {
		reader := bytes.NewReader(plots[0])
		pushoverMessage.AddAttachment(reader) //nolint
	}

	return pushoverMessage
}

func (client *pushoverClient) buildMessage(events moira.NotificationEvents, throttled bool) string {
	var message bytes.Buffer
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}

		message.WriteString(fmt.Sprintf("%s: %s = %s (%s to %s)", event.FormatTimestamp(client.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State))
		if msg := event.CreateMessage(client.location); len(msg) > 0 {
			message.WriteString(fmt.Sprintf(". %s\n", msg))
		} else {
			message.WriteString("\n")
		}
	}

	if len(events) > printEventsCount {
		message.WriteString(fmt.Sprintf("\n...and %d more events.", len(events)-printEventsCount))
	}

	if throttled {
		message.WriteString("\nPlease, fix your system or tune this trigger to generate less events.")
	}

	return message.String()
}

func (client *pushoverClient) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, trigger.GetTags(), len(events))
	tags := 1

	for len([]rune(title)) > titleLimit {
		var tagBuffer bytes.Buffer
		for i := 0; i < len(trigger.Tags)-tags; i++ {
			tagBuffer.WriteString(fmt.Sprintf("[%s]", trigger.Tags[i]))
		}

		title = fmt.Sprintf("%s %s %s.... (%d)", state, trigger.Name, tagBuffer.String(), len(events))
		tags++
	}

	return title
}

func (client *pushoverClient) getMessagePriority(events moira.NotificationEvents) int {
	priority := pushover_client.PriorityNormal
	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			priority = pushover_client.PriorityEmergency
		}

		if priority != pushover_client.PriorityEmergency && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			priority = pushover_client.PriorityHigh
		}
	}

	return priority
}
