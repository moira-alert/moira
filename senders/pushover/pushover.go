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
type pushover struct {
	APIToken string `mapstructure:"api_token"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface via pushover
type Sender struct {
	logger   moira.Logger
	location *time.Location
	client   *pushover_client.Pushover

	apiToken string
	frontURI string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var p pushover
	err := mapstructure.Decode(senderSettings, &p)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to pushover config: %w", err)
	}
	sender.apiToken = p.APIToken
	if sender.apiToken == "" {
		return fmt.Errorf("can not read pushover api_token from config")
	}
	sender.client = pushover_client.New(sender.apiToken)
	sender.logger = logger
	sender.frontURI = p.FrontURI
	sender.location = location
	return nil
}

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	pushoverMessage := sender.makePushoverMessage(events, trigger, plots, throttled)

	sender.logger.Debug().
		String("message_title", pushoverMessage.Title).
		String("message", pushoverMessage.Message).
		Msg("Calling pushover with message title")

	recipient := pushover_client.NewRecipient(contact.Value)
	_, err := sender.client.SendMessage(pushoverMessage, recipient)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
	}
	return nil
}

func (sender *Sender) makePushoverMessage(events moira.NotificationEvents, trigger moira.TriggerData, plots [][]byte, throttled bool) *pushover_client.Message {
	pushoverMessage := &pushover_client.Message{
		Message:   sender.buildMessage(events, throttled),
		Title:     sender.buildTitle(events, trigger),
		Priority:  sender.getMessagePriority(events),
		Retry:     5 * time.Minute, //nolint
		Expire:    time.Hour,
		Timestamp: events[len(events)-1].Timestamp,
	}
	url := trigger.GetTriggerURI(sender.frontURI)
	if len(url) < urlLimit {
		pushoverMessage.URL = url
	}
	if len(plots) > 0 {
		reader := bytes.NewReader(plots[0])
		pushoverMessage.AddAttachment(reader) //nolint
	}

	return pushoverMessage
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, throttled bool) string {
	var message bytes.Buffer
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}
		message.WriteString(fmt.Sprintf("%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(false), event.OldState, event.State))
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
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

func (sender *Sender) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData) string {
	title := fmt.Sprintf("%s %s %s (%d)", events.GetSubjectState(), trigger.Name, trigger.GetTags(), len(events))
	tags := 1
	for len([]rune(title)) > titleLimit {
		var tagBuffer bytes.Buffer
		for i := 0; i < len(trigger.Tags)-tags; i++ {
			tagBuffer.WriteString(fmt.Sprintf("[%s]", trigger.Tags[i]))
		}
		title = fmt.Sprintf("%s %s %s.... (%d)", events.GetSubjectState(), trigger.Name, tagBuffer.String(), len(events))
		tags++
	}
	return title
}

func (sender *Sender) getMessagePriority(events moira.NotificationEvents) int {
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
