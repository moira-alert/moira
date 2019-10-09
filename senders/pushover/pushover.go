package pushover

import (
	"bytes"
	"fmt"
	"time"

	"github.com/moira-alert/moira"

	"github.com/gregdel/pushover"
)

const printEventsCount int = 5
const titleLimit = 250
const urlLimit = 512

// Sender implements moira sender interface via pushover
type Sender struct {
	logger   moira.Logger
	location *time.Location
	client   *pushover.Pushover

	apiToken string
	frontURI string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.apiToken = senderSettings["api_token"]
	if sender.apiToken == "" {
		return fmt.Errorf("can not read pushover api_token from config")
	}
	sender.client = pushover.New(sender.apiToken)
	sender.logger = logger
	sender.frontURI = senderSettings["front_uri"]
	sender.location = location
	return nil
}

// SendEvents implements pushover build and send message functionality
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	pushoverMessage := sender.makePushoverMessage(events, contact, trigger, plots, throttled)

	sender.logger.Debugf("Calling pushover with message title %s, body %s", pushoverMessage.Title, pushoverMessage.Message)
	recipient := pushover.NewRecipient(contact.Value)
	_, err := sender.client.SendMessage(pushoverMessage, recipient)
	if err != nil {
		return fmt.Errorf("failed to send %s event message to pushover user %s: %s", trigger.ID, contact.Value, err.Error())
	}
	return nil
}

func (sender *Sender) makePushoverMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) *pushover.Message {
	pushoverMessage := &pushover.Message{
		Message:   sender.buildMessage(events, throttled),
		Title:     sender.buildTitle(events, trigger),
		Priority:  sender.getMessagePriority(events),
		Retry:     5 * time.Minute,
		Expire:    time.Hour,
		Timestamp: events[len(events)-1].Timestamp,
	}
	url := trigger.GetTriggerURI(sender.frontURI)
	if len(url) < urlLimit {
		pushoverMessage.URL = url
	}
	if len(plots) > 0 {
		reader := bytes.NewReader(plots[0])
		pushoverMessage.AddAttachment(reader)
	}

	return pushoverMessage
}

func (sender *Sender) buildMessage(events moira.NotificationEvents, throttled bool) string {
	var message bytes.Buffer
	for i, event := range events {
		if i > printEventsCount-1 {
			break
		}
		message.WriteString(fmt.Sprintf("%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricsValues(), event.OldState, event.State))
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
	priority := pushover.PriorityNormal
	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			priority = pushover.PriorityEmergency
		}
		if priority != pushover.PriorityEmergency && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			priority = pushover.PriorityHigh
		}
	}
	return priority
}
